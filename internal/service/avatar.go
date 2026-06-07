package service

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/coolycow/gophprofile/internal/config"
	profileError "github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/logger"
	"github.com/coolycow/gophprofile/internal/model"
	"github.com/coolycow/gophprofile/internal/observability"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/minio/minio-go/v7"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// AvatarJobPublisher асинхронно ставит задание на обработку аватара в RabbitMQ.
type AvatarJobPublisher interface {
	PublishAvatarProcessingJob(ctx context.Context, avatarID string) error
	PublishAvatarDeletionByS3KeyJob(ctx context.Context, s3Key string, thumbnailS3Keys []string) error
}

// AvatarService Сервис для работы с аватарами
type AvatarService interface {
	UploadAvatar(ctx context.Context, userID string, file io.Reader, fileHeader *multipart.FileHeader) (*model.Avatar, error)

	GetAvatarByID(ctx context.Context, avatarID string) (*model.Avatar, error)
	GetAvatarByUserID(ctx context.Context, userID string) (*model.Avatar, error)

	DeleteAvatarByID(ctx context.Context, callerUserID, avatarID string) error
	DeleteAvatarByUserID(ctx context.Context, callerUserID, pathUserID string) error

	GetUserAvatars(ctx context.Context, userID string) ([]*model.Avatar, error)

	// OpenAvatarObject открывает объект в MinIO по ключу из строки avatar (вызывающий обязан Close()).
	OpenAvatarObject(ctx context.Context, avatar *model.Avatar) (*minio.Object, error)

	// PrepareAvatarDownload готовит тело ответа GET с учётом size/format (оригинал или миниатюра, перекодирование jpeg/png).
	PrepareAvatarDownload(ctx context.Context, avatar *model.Avatar, size, format string) (*AvatarDownload, error)
}

// Реализация сервисного слоя
type avatarService struct {
	repo         repository.GophProfileRepository
	cfg          *config.ConfigServer
	minioClient  *minio.Client
	jobPublisher AvatarJobPublisher
}

// NewAvatarService инициализация сервиса. jobPublisher может быть nil — тогда задания в очередь не отправляются.
func NewAvatarService(cfg *config.ConfigServer, repo repository.GophProfileRepository, minioClient *minio.Client, jobPublisher AvatarJobPublisher) AvatarService {
	return &avatarService{
		repo:         repo,
		cfg:          cfg,
		minioClient:  minioClient,
		jobPublisher: jobPublisher,
	}
}

// UploadAvatar загружает аватарку
func (s *avatarService) UploadAvatar(ctx context.Context, userID string, file io.Reader, fileHeader *multipart.FileHeader) (*model.Avatar, error) {
	start := time.Now()
	status := "error"
	var uploadedSize int64
	defer func() {
		observability.ObserveUpload(userID, status, time.Since(start), uploadedSize)
	}()

	ctx, span := otel.Tracer(observability.Tracer()).Start(ctx, "upload_avatar")
	defer span.End()
	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("file_name", fileHeader.Filename),
	)

	// Получаем максимальный размер файла
	maxB := s.cfg.MaxFileSize
	if maxB < 1 {
		maxB = 1 << 20
	}

	// Получаем Content-Type из заголовка файла
	rawCT := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if rawCT == "" {
		return nil, profileError.CustomError{
			Message:    "Invalid file format",
			Details:    "Supported formats: jpeg, png, webp",
			StatusCode: http.StatusBadRequest,
		}
	}

	// Парсим Content-Type из заголовка файла
	mediaType, _, err := mime.ParseMediaType(rawCT)
	if err != nil {
		return nil, profileError.CustomError{
			Message:    "Invalid file format",
			Details:    "Supported formats: jpeg, png, webp",
			StatusCode: http.StatusBadRequest,
		}
	}

	// Проверяем, поддерживается ли Content-Type
	switch mediaType {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return nil, profileError.CustomError{
			Message:    "Invalid file format",
			Details:    "Supported formats: jpeg, png, webp",
			StatusCode: http.StatusBadRequest,
		}
	}

	// Читаем файл с ограничением размера
	limited := io.LimitReader(file, int64(maxB)+1)
	reader, err := io.ReadAll(limited)
	if err != nil {
		return nil, profileError.CustomError{
			Message:    fmt.Sprintf("failed to read file: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Проверяем, пустой ли файл
	if len(reader) == 0 {
		return nil, profileError.CustomError{
			Message:    "Invalid file format",
			Details:    "empty file",
			StatusCode: http.StatusBadRequest,
		}
	}

	// Проверяем, не превышает ли размер файла максимальный размер
	if len(reader) > maxB {
		return nil, profileError.CustomError{
			Message:    "File too large",
			StatusCode: http.StatusRequestEntityTooLarge,
			Meta:       map[string]any{"max_size": int64(maxB)},
		}
	}

	// Сохраняем файл в хранилище MinIO
	bucketName := s.cfg.MinioBucketName
	objectName := fmt.Sprintf("%s/%s", userID, fileHeader.Filename)
	contentType := mediaType
	n := int64(len(reader))
	span.SetAttributes(attribute.Int64("file_size", n))

	logger.FromContext(ctx).Info("uploading avatar",
		"user_id", userID,
		"file_size", n,
		"mime_type", mediaType,
	)

	info, err := s.putObject(ctx, bucketName, objectName, bytes.NewReader(reader), n, contentType)

	// Если ошибка, возвращаем ошибку
	if err != nil {
		return nil, profileError.CustomError{
			Message:    fmt.Sprintf("failed to upload file to MinIO: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Текущая аватарка (если есть) — заменяем при новой загрузке
	currentAvatar, err := s.repo.GetAvatarByUserID(ctx, userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	// ID текущей аватарки (если есть)
	currentAvatarID := ""
	if currentAvatar != nil {
		currentAvatarID = currentAvatar.ID
	}

	// Сохраняем новую аватарку в базу данных
	avatar, err := s.repo.UploadAvatar(ctx, userID, fileHeader.Filename, contentType, info.Size, info.Key, "[]", "completed", "pending", currentAvatarID)
	if err != nil {
		// Отправляем задание на удаление загруженного файла из S3
		if s.jobPublisher != nil && info.Key != "" {
			if pubErr := s.jobPublisher.PublishAvatarDeletionByS3KeyJob(ctx, info.Key, nil); pubErr != nil {
				return nil, profileError.CustomError{
					Message:    fmt.Sprintf("failed to enqueue new avatar deletion: %s", pubErr),
					StatusCode: http.StatusInternalServerError,
				}
			}
		}
		return nil, err
	}

	// Если publisher не nil, отправляем задание на обработку аватарки
	if s.jobPublisher != nil {
		// Старая аватарка уже снята с записи в БД внутри UploadAvatar; ставим задачу на очистку S3 и т.п.
		if currentAvatar != nil && currentAvatar.S3Key != avatar.S3Key {
			logger.Log.Info("Enqueuing old avatar deletion", "s3_key", currentAvatar.S3Key)
			if pubErr := s.jobPublisher.PublishAvatarDeletionByS3KeyJob(ctx, currentAvatar.S3Key, []string(currentAvatar.ThumbnailS3Keys)); pubErr != nil {
				return nil, profileError.CustomError{
					Message:    fmt.Sprintf("failed to enqueue old avatar deletion: %s", pubErr),
					StatusCode: http.StatusInternalServerError,
				}
			}
		}

		// Отправляем задание на обработку новой аватарки
		if pubErr := s.jobPublisher.PublishAvatarProcessingJob(ctx, avatar.ID); pubErr != nil {
			return nil, profileError.CustomError{
				Message:    fmt.Sprintf("failed to enqueue new avatar processing: %s", pubErr),
				StatusCode: http.StatusInternalServerError,
			}
		}
	}

	status = "success"
	uploadedSize = info.Size
	return avatar, nil
}

// GetAvatarByID получает аватарку по ID
func (s *avatarService) GetAvatarByID(ctx context.Context, avatarID string) (*model.Avatar, error) {
	return s.repo.GetAvatarByID(ctx, avatarID)
}

// GetAvatarByUserID получает аватарку по ID пользователя
func (s *avatarService) GetAvatarByUserID(ctx context.Context, userID string) (*model.Avatar, error) {
	return s.repo.GetAvatarByUserID(ctx, userID)
}

// OpenAvatarObject открывает поток объекта в MinIO для записи из БД.
func (s *avatarService) OpenAvatarObject(ctx context.Context, avatar *model.Avatar) (*minio.Object, error) {
	if avatar == nil || strings.TrimSpace(avatar.S3Key) == "" {
		return nil, fmt.Errorf("avatar or s3 key is empty")
	}
	return s.getObject(ctx, s.cfg.MinioBucketName, avatar.S3Key)
}

// DeleteAvatarByID удаляет аватарку по ID, если callerUserID совпадает с владельцем.
func (s *avatarService) DeleteAvatarByID(ctx context.Context, callerUserID, avatarID string) error {
	callerUserID = strings.TrimSpace(callerUserID)
	avatarID = strings.TrimSpace(avatarID)

	// Получаем аватарку по ID
	avatar, err := s.repo.GetAvatarByID(ctx, avatarID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return profileError.CustomError{
				Message:    "avatar not found",
				StatusCode: http.StatusNotFound,
			}
		}
		return err
	}

	// Проверяем, совпадает ли ID владельца аватарки с ID пользователя, который пытается удалить аватарку
	if !strings.EqualFold(strings.TrimSpace(avatar.UserID), callerUserID) {
		return profileError.CustomError{
			Message:    "Forbidden",
			Details:    "You can only delete your own avatars",
			StatusCode: http.StatusForbidden,
		}
	}

	// Удаляем аватарку по ID из базы данных
	err = s.repo.DeleteAvatarByID(ctx, avatarID)
	if err != nil {
		return err
	}

	// Если publisher не nil, отправляем задание на удаление аватарки из хранилища
	if s.jobPublisher != nil {
		if pubErr := s.jobPublisher.PublishAvatarDeletionByS3KeyJob(ctx, avatar.S3Key, []string(avatar.ThumbnailS3Keys)); pubErr != nil {
			return profileError.CustomError{
				Message:    fmt.Sprintf("failed to enqueue avatar deletion: %s", pubErr),
				StatusCode: http.StatusInternalServerError,
			}
		}
	}

	return nil
}

// DeleteAvatarByUserID удаляет аватар пользователя; pathUserID должен совпадать с callerUserID.
func (s *avatarService) DeleteAvatarByUserID(ctx context.Context, callerUserID, pathUserID string) error {
	callerUserID = strings.TrimSpace(callerUserID)
	pathUserID = strings.TrimSpace(pathUserID)

	// Проверяем, совпадает ли ID владельца аватарки с ID пользователя, который пытается удалить аватарку
	if !strings.EqualFold(callerUserID, pathUserID) {
		return profileError.CustomError{
			Message:    "Forbidden",
			Details:    "You can only delete your own avatars",
			StatusCode: http.StatusForbidden,
		}
	}

	// Получаем аватарку по ID пользователя
	avatar, err := s.repo.GetAvatarByUserID(ctx, pathUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return profileError.CustomError{
				Message:    "avatar not found",
				StatusCode: http.StatusNotFound,
			}
		}
		return err
	}

	// Удаляем аватарку по ID пользователя из базы данных
	err = s.repo.DeleteAvatarByUserID(ctx, pathUserID)
	if err != nil {
		return err
	}

	// Если publisher не nil, отправляем задание на удаление аватарки из хранилища
	if s.jobPublisher != nil {
		if pubErr := s.jobPublisher.PublishAvatarDeletionByS3KeyJob(ctx, avatar.S3Key, []string(avatar.ThumbnailS3Keys)); pubErr != nil {
			return profileError.CustomError{
				Message:    fmt.Sprintf("failed to enqueue avatar deletion: %s", pubErr),
				StatusCode: http.StatusInternalServerError,
			}
		}
	}

	return nil
}

// GetUserAvatars получает список аватарок пользователя
func (s *avatarService) GetUserAvatars(ctx context.Context, userID string) ([]*model.Avatar, error) {
	return s.repo.GetUserAvatars(ctx, userID)
}
