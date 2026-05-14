package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strings"

	profileError "github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/model"
	"github.com/minio/minio-go/v7"
	webp "github.com/skrashevich/go-webp"
)

// AvatarDownload тело и заголовки для GET аватарки.
type AvatarDownload struct {
	Body          io.ReadCloser // тело ответа
	ContentType   string        // Content-Type
	ContentLength int64         // Content-Length
	ETag          string        // ETag
}

// normalizeAvatarSizeParam: пустое значение и "original" — полный оригинал; иначе миниатюра 100x100 / 300x300.
func normalizeAvatarSizeParam(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "", "original":
		return "original"
	case "100x100", "300x300":
		return s
	default:
		return "_invalid_"
	}
}

// normalizeAvatarFormatParam нормализует параметр format
func normalizeAvatarFormatParam(f string) string {
	f = strings.ToLower(strings.TrimSpace(f))
	switch f {
	case "":
		return ""
	case "jpg", "jpeg":
		return "jpeg"
	case "png":
		return "png"
	case "webp":
		return "webp"
	default:
		return "_invalid_"
	}
}

// thumbnailSideFromSizeLabel возвращает размер миниатюры из метки размера
func thumbnailSideFromSizeLabel(size string) int {
	switch size {
	case "100x100":
		return 100
	case "300x300":
		return 300
	default:
		return 0
	}
}

// thumbnailObjectKeyGuess генерирует ключ миниатюры
func (s *avatarService) thumbnailObjectKeyGuess(a *model.Avatar, side int) string {
	return fmt.Sprintf("%s/thumbnails/%s_%d.jpg", strings.TrimSpace(a.UserID), strings.TrimSpace(a.ID), side)
}

// resolveThumbnailKey разрешает ключ миниатюры
func (s *avatarService) resolveThumbnailKey(ctx context.Context, a *model.Avatar, side int) (string, error) {
	want := fmt.Sprintf("_%d.jpg", side)
	for _, k := range []string(a.ThumbnailS3Keys) {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if strings.HasSuffix(k, want) {
			return k, nil
		}
	}
	candidate := s.thumbnailObjectKeyGuess(a, side)
	_, err := s.minioClient.StatObject(ctx, s.cfg.MinioBucketName, candidate, minio.StatObjectOptions{})
	if err == nil {
		return candidate, nil
	}
	return "", profileError.CustomError{
		Message:    "Avatar not found",
		Details:    "thumbnail is not ready yet",
		StatusCode: http.StatusNotFound,
	}
}

// readObjectLimited читает объект из MinIO с ограничением размера
func (s *avatarService) readObjectLimited(ctx context.Context, objectKey string) ([]byte, string, error) {
	maxB := s.cfg.MaxFileSize
	if maxB < 1 {
		maxB = 1 << 20
	}
	obj, err := s.minioClient.GetObject(ctx, s.cfg.MinioBucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", err
	}
	defer obj.Close()

	stat, err := obj.Stat()
	if err != nil {
		return nil, "", err
	}

	limited := io.LimitReader(obj, int64(maxB)+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", err
	}
	if len(raw) > maxB {
		return nil, "", fmt.Errorf("object exceeds max size (%d bytes)", maxB)
	}
	ct := strings.TrimSpace(stat.ContentType)
	return raw, ct, nil
}

// etagFromBytes генерирует ETag из байтов
func etagFromBytes(b []byte) string {
	h := sha256.Sum256(b)
	return `"` + hex.EncodeToString(h[:]) + `"`
}

// encodeImageToFormat кодирует изображение в формат
func encodeImageToFormat(img image.Image, format string) ([]byte, string, error) {
	switch format {
	case "jpeg":
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/jpeg", nil
	case "png":
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/png", nil
	case "webp":
		var buf bytes.Buffer
		if err := webp.Encode(&buf, img, &webp.Options{Lossy: true, Quality: 85}); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/webp", nil
	default:
		return nil, "", fmt.Errorf("unsupported output format %q", format)
	}
}

// PrepareAvatarDownload готовит поток для GET /avatars/:id и GET /users/:id/avatar.
// size: не указан или original — оригинал; 100x100 / 300x300 — миниатюра.
// format: не указан — отдача объекта из хранилища как есть; jpeg|png|webp — перекодирование.
func (s *avatarService) PrepareAvatarDownload(ctx context.Context, avatar *model.Avatar, sizeParam, formatParam string) (*AvatarDownload, error) {
	if avatar == nil {
		return nil, profileError.CustomError{Message: "Avatar not found", StatusCode: http.StatusNotFound}
	}

	// Нормализуем параметр size
	size := normalizeAvatarSizeParam(sizeParam)
	if size == "_invalid_" {
		return nil, profileError.CustomError{
			Message:    "Invalid query parameter",
			Details:    `size must be one of: original, 100x100, 300x300`,
			StatusCode: http.StatusBadRequest,
		}
	}

	// Нормализуем параметр format
	format := normalizeAvatarFormatParam(formatParam)
	if format == "_invalid_" {
		return nil, profileError.CustomError{
			Message:    "Invalid query parameter",
			Details:    `format must be one of: jpeg, png, webp`,
			StatusCode: http.StatusBadRequest,
		}
	}

	var objectKey string
	var nativeCT string

	// Если size == "original", то используем оригинальный объект
	if size == "original" {
		objectKey = strings.TrimSpace(avatar.S3Key)
		nativeCT = strings.TrimSpace(avatar.MimeType)
	} else {
		side := thumbnailSideFromSizeLabel(size)
		k, err := s.resolveThumbnailKey(ctx, avatar, side)
		if err != nil {
			return nil, err
		}
		objectKey = k
		nativeCT = "image/jpeg"
	}

	// Если объект не найден, то возвращаем ошибку
	if objectKey == "" {
		return nil, profileError.CustomError{Message: "Avatar not found", StatusCode: http.StatusNotFound}
	}

	// Без format — поток из MinIO (оригинальный Content-Type / ETag объекта).
	if format == "" {
		// Получаем объект из MinIO
		obj, err := s.minioClient.GetObject(ctx, s.cfg.MinioBucketName, objectKey, minio.GetObjectOptions{})
		if err != nil {
			if minio.ToErrorResponse(err).Code == "NoSuchKey" {
				return nil, profileError.CustomError{Message: "Avatar not found", StatusCode: http.StatusNotFound}
			}
			return nil, err
		}

		// Получаем статистику объекта
		stat, err := obj.Stat()
		if err != nil {
			obj.Close()
			if minio.ToErrorResponse(err).Code == "NoSuchKey" {
				return nil, profileError.CustomError{Message: "Avatar not found", StatusCode: http.StatusNotFound}
			}
			return nil, err
		}

		// Получаем Content-Type из статистики объекта
		ct := nativeCT
		if strings.TrimSpace(stat.ContentType) != "" {
			ct = strings.TrimSpace(stat.ContentType)
		}
		if ct == "" {
			ct = "application/octet-stream"
		}

		// Получаем ETag из статистики объекта
		etag := strings.TrimSpace(stat.ETag)
		if etag != "" && !strings.HasPrefix(etag, `"`) {
			etag = `"` + strings.Trim(etag, `"`) + `"`
		}

		// Возвращаем поток для GET /avatars/:id и GET /users/:id/avatar
		return &AvatarDownload{
			Body:          obj,
			ContentType:   ct,
			ContentLength: stat.Size,
			ETag:          etag,
		}, nil
	}

	// С format — перекодирование (в т.ч. webp для миниатюр jpeg).
	raw, _, err := s.readObjectLimited(ctx, objectKey)
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, profileError.CustomError{Message: "Avatar not found", StatusCode: http.StatusNotFound}
		}
		return nil, err
	}

	// Декодируем изображение
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, profileError.CustomError{
			Message:    "failed to decode image",
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Кодируем изображение в формат
	out, ct, err := encodeImageToFormat(img, format)
	if err != nil {
		return nil, err
	}

	// Возвращаем поток для GET /avatars/:id и GET /users/:id/avatar
	return &AvatarDownload{
		Body:          io.NopCloser(bytes.NewReader(out)),
		ContentType:   ct,
		ContentLength: int64(len(out)),
		ETag:          etagFromBytes(out),
	}, nil
}
