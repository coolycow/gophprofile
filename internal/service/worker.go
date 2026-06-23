package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"math"
	"strings"
	"time"

	"github.com/coolycow/gophprofile/internal/config"
	"github.com/coolycow/gophprofile/internal/model"
	"github.com/coolycow/gophprofile/internal/observability"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/minio/minio-go/v7"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/image/draw"

	_ "image/png"

	_ "golang.org/x/image/webp"
)

const (
	avatarThumbSize100 = 100
	avatarThumbSize300 = 300

	processingStatusCompleted = "completed"
	processingStatusFailed    = "failed"
)

// WorkerService операции воркера по сообщениям из очереди.
type WorkerService interface {
	ProcessAvatar(ctx context.Context, avatarID string) error
	DeleteAvatarByS3Key(ctx context.Context, s3Key string, thumbnailS3Keys []string) error
}

// Реализация сервисного слоя
type workerService struct {
	repo         repository.GophProfileRepository
	cfg          *config.ConfigWorker
	minioClient  *minio.Client
	minioBreaker *gobreaker.CircuitBreaker
}

// NewWorkerService создаёт сервис воркера (асинхронная обработка после HTTP).
func NewWorkerService(repo repository.GophProfileRepository, cfg *config.ConfigWorker, minioClient *minio.Client, minioBreaker *gobreaker.CircuitBreaker) WorkerService {
	return &workerService{
		repo:         repo,
		cfg:          cfg,
		minioClient:  minioClient,
		minioBreaker: minioBreaker,
	}
}

// ProcessAvatar обрабатывает аватарку
func (s *workerService) ProcessAvatar(ctx context.Context, avatarID string) error {
	start := time.Now()
	jobType := "process"
	status := "error"
	defer func() {
		observability.ObserveJob(jobType, status, time.Since(start))
	}()

	ctx, span := otel.Tracer(observability.Tracer()).Start(ctx, "worker.process_avatar")
	defer span.End()
	span.SetAttributes(attribute.String("avatar_id", avatarID))

	// Обрезаем пробелы
	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return fmt.Errorf("avatar id is empty")
	}

	// Получаем аватарку по ID
	avatar, err := s.repo.GetAvatarByID(ctx, avatarID)
	if err != nil {
		return err
	}

	// Если аватар уже обработан и есть миниатюры, то выходим
	if avatar.ProcessingStatus == processingStatusCompleted && len(avatar.ThumbnailS3Keys) >= 2 {
		return nil
	}

	// Читаем оригинальный аватар
	raw, err := s.readOriginalAvatar(ctx, avatar.S3Key)
	if err != nil {
		_ = s.repo.UpdateAvatarThumbnails(ctx, avatarID, nil, processingStatusFailed, nil)
		return err
	}

	// Декодируем изображение
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		_ = s.repo.UpdateAvatarThumbnails(ctx, avatarID, nil, processingStatusFailed, nil)
		return fmt.Errorf("decode avatar image: %w", err)
	}

	// Генерируем ключи миниатюр
	sizes := []int{avatarThumbSize100, avatarThumbSize300}
	var keys []string

	// Генерируем миниатюры
	for _, side := range sizes {
		thumb := squareCoverThumbnail(img, side)
		jpegBuf, encErr := encodeJPEG(thumb, 85)
		if encErr != nil {
			s.removeMinioKeys(ctx, keys)
			_ = s.repo.UpdateAvatarThumbnails(ctx, avatarID, nil, processingStatusFailed, nil)
			return encErr
		}

		key := thumbnailObjectKey(avatar, side)
		_, putErr := s.putObject(ctx, s.cfg.MinioBucketName, key, bytes.NewReader(jpegBuf), int64(len(jpegBuf)), "image/jpeg")
		if putErr != nil {
			s.removeMinioKeys(ctx, keys)
			_ = s.repo.UpdateAvatarThumbnails(ctx, avatarID, nil, processingStatusFailed, nil)
			return putErr
		}
		keys = append(keys, key)
	}

	bounds := img.Bounds()
	dims := &model.ImageDimensions{Width: bounds.Dx(), Height: bounds.Dy()}

	// Обновляем ключи миниатюр в S3 и статус постобработки
	if err := s.repo.UpdateAvatarThumbnails(ctx, avatarID, keys, processingStatusCompleted, dims); err != nil {
		s.removeMinioKeys(ctx, keys)
		return err
	}

	status = "success"
	return nil
}

// thumbnailObjectKey генерирует ключ миниатюры
func thumbnailObjectKey(a *model.Avatar, side int) string {
	return fmt.Sprintf("%s/thumbnails/%s_%d.jpg", strings.TrimSpace(a.UserID), strings.TrimSpace(a.ID), side)
}

// readOriginalAvatar читает оригинальный аватар из S3
func (s *workerService) readOriginalAvatar(ctx context.Context, s3Key string) ([]byte, error) {
	// Обрезаем пробелы
	s3Key = strings.TrimSpace(s3Key)
	if s3Key == "" {
		return nil, fmt.Errorf("s3 key is empty")
	}

	// Получаем оригинальный аватар из S3
	obj, err := s.getObject(ctx, s.cfg.MinioBucketName, s3Key)
	if err != nil {
		return nil, err
	}
	defer obj.Close()

	// Ограничиваем размер файла
	maxBytes := s.cfg.MaxFileSize
	if maxBytes < 1 {
		maxBytes = 1 << 20 // fallback if misconfigured
	}

	// Читаем оригинальный аватар из S3
	limited := io.LimitReader(obj, int64(maxBytes)+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}

	// Если размер файла превышает максимальный размер, то возвращаем ошибку
	if len(raw) > maxBytes {
		return nil, fmt.Errorf("avatar object exceeds max size (%d bytes)", maxBytes)
	}

	return raw, nil
}

// squareCoverThumbnail генерирует квадратную миниатюру
func squareCoverThumbnail(src image.Image, side int) image.Image {
	if side < 1 {
		side = 1
	}
	b := src.Bounds()
	sw, sh := float64(b.Dx()), float64(b.Dy())
	if sw < 1 || sh < 1 {
		dst := image.NewRGBA(image.Rect(0, 0, side, side))
		return dst
	}

	scale := math.Max(float64(side)/sw, float64(side)/sh)
	nw := int(math.Round(sw * scale))
	nh := int(math.Round(sh * scale))
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}

	tmp := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.BiLinear.Scale(tmp, tmp.Bounds(), src, b, draw.Over, nil)

	x0 := (nw - side) / 2
	y0 := (nh - side) / 2
	out := image.NewRGBA(image.Rect(0, 0, side, side))
	srcRect := image.Rect(x0, y0, x0+side, y0+side)
	draw.Draw(out, out.Bounds(), tmp, srcRect.Min, draw.Src)
	return out
}

// encodeJPEG кодирует изображение в JPEG
func encodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// removeMinioKeys удаляет ключи миниатюр из S3
func (s *workerService) removeMinioKeys(ctx context.Context, keys []string) {
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		_ = s.removeObject(ctx, s.cfg.MinioBucketName, k)
	}
}

// mergeS3DeletionKeys объединяет ключ оригинала и миниатюр без дубликатов и пустых строк (оригинал первым).
func mergeS3DeletionKeys(main string, thumbnails []string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(k string) {
		k = strings.TrimSpace(k)
		if k == "" {
			return
		}
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	add(main)
	for _, t := range thumbnails {
		add(t)
	}
	return out
}

// DeleteAvatarByS3Key удаляет из S3 оригинал и все переданные миниатюры.
func (s *workerService) DeleteAvatarByS3Key(ctx context.Context, s3Key string, thumbnailS3Keys []string) error {
	start := time.Now()
	jobType := "delete_by_s3_key"
	status := "error"
	defer func() {
		observability.ObserveJob(jobType, status, time.Since(start))
	}()

	ctx, span := otel.Tracer(observability.Tracer()).Start(ctx, "worker.delete_by_s3_key")
	defer span.End()
	span.SetAttributes(attribute.String("s3_key", s3Key))

	// Объединяем ключи оригинального аватара и миниатюр без дубликатов и пустых строк (оригинал первым)
	keys := mergeS3DeletionKeys(s3Key, thumbnailS3Keys)
	var firstErr error

	// Удаляем оригинал и все миниатюры
	for _, k := range keys {
		err := s.removeObject(ctx, s.cfg.MinioBucketName, k)
		if err != nil && firstErr == nil {
			firstErr = err
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	}

	if firstErr == nil {
		status = "success"
	}
	return firstErr
}
