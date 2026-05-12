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
	"strconv"
	"strings"

	"github.com/coolycow/gophprofile/internal/config"
	profileError "github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/logger"
	"github.com/coolycow/gophprofile/internal/model"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/go-playground/validator/v10"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

// AvatarJobPublisher асинхронно ставит задание на обработку аватара в RabbitMQ.
type AvatarJobPublisher interface {
	PublishAvatarProcessingJob(ctx context.Context, avatarID string) error
	PublishAvatarDeletionByS3KeyJob(ctx context.Context, s3Key string) error
}

// AvatarService Сервис для работы с аватарами
type AvatarService interface {
	UploadAvatar(ctx context.Context, userID string, file *multipart.File, fileHeader *multipart.FileHeader) (*model.Avatar, error)

	GetAvatarByID(ctx context.Context, avatarID string) (*model.Avatar, error)
	GetAvatarByUserID(ctx context.Context, userID string) (*model.Avatar, error)

	DeleteAvatarByID(ctx context.Context, callerUserID, avatarID string) error
	DeleteAvatarByUserID(ctx context.Context, callerUserID, pathUserID string) error

	GetUserAvatars(ctx context.Context, userID string) ([]*model.Avatar, error)

	// OpenAvatarObject открывает объект в MinIO по ключу из строки avatar (вызывающий обязан Close()).
	OpenAvatarObject(ctx context.Context, avatar *model.Avatar) (*minio.Object, error)
}

// Реализация сервисного слоя
type avatarService struct {
	repo         repository.GophProfileRepository
	cfg          *config.ConfigServer
	validator    *validator.Validate
	minioClient  *minio.Client
	jobPublisher AvatarJobPublisher
}

// NewAvatarService инициализация сервиса. jobPublisher может быть nil — тогда задания в очередь не отправляются.
func NewAvatarService(cfg *config.ConfigServer, repo repository.GophProfileRepository, minioClient *minio.Client, jobPublisher AvatarJobPublisher) AvatarService {
	return &avatarService{
		repo:         repo,
		cfg:          cfg,
		validator:    validator.New(),
		minioClient:  minioClient,
		jobPublisher: jobPublisher,
	}
}

// UploadAvatar загружает аватарку
func (s *avatarService) UploadAvatar(ctx context.Context, userID string, file *multipart.File, fileHeader *multipart.FileHeader) (*model.Avatar, error) {
	// Валидация файла
	if err := s.validator.Var(fileHeader.Size, "required,min=1,max="+strconv.Itoa(s.cfg.MaxFileSize)); err != nil {
		return nil, profileError.CustomError{
			Message:    fmt.Sprintf("file size validation failed: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	// Валидация MIME (validator не регистрирует тег mime по умолчанию — используем net/mime)
	rawCT := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if rawCT == "" {
		return nil, profileError.CustomError{
			Message:    "content type is required",
			StatusCode: http.StatusBadRequest,
		}
	}

	// Парсим MIME тип
	mediaType, _, err := mime.ParseMediaType(rawCT)
	if err != nil {
		return nil, profileError.CustomError{
			Message:    fmt.Sprintf("invalid content type: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	// Проверяем, является ли MIME тип изображением
	// Если тип соответствует, то продолжаем
	// Если тип не соответствует, то возвращаем ошибку
	switch mediaType {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return nil, profileError.CustomError{
			Message:    fmt.Sprintf("mime type not allowed: %s (allowed: image/jpeg, image/png, image/webp)", mediaType),
			StatusCode: http.StatusBadRequest,
		}
	}

	// Читаем файл в память
	reader, err := io.ReadAll(*file)
	if err != nil {
		return nil, profileError.CustomError{
			Message:    fmt.Sprintf("failed to open file: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Сохраняем файл в хранилище MinIO
	bucketName := s.cfg.MinioBucketName
	objectName := fmt.Sprintf("%s/%s", userID, fileHeader.Filename)
	contentType := mediaType

	info, err := s.minioClient.PutObject(ctx, bucketName, objectName, bytes.NewReader(reader), fileHeader.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})

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
			if pubErr := s.jobPublisher.PublishAvatarDeletionByS3KeyJob(ctx, info.Key); pubErr != nil {
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
			logger.Log.Info("Enqueuing old avatar deletion", zap.String("s3_key", currentAvatar.S3Key))
			if pubErr := s.jobPublisher.PublishAvatarDeletionByS3KeyJob(ctx, currentAvatar.S3Key); pubErr != nil {
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
	return s.minioClient.GetObject(ctx, s.cfg.MinioBucketName, avatar.S3Key, minio.GetObjectOptions{})
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
		if pubErr := s.jobPublisher.PublishAvatarDeletionByS3KeyJob(ctx, avatar.S3Key); pubErr != nil {
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
		if pubErr := s.jobPublisher.PublishAvatarDeletionByS3KeyJob(ctx, avatar.S3Key); pubErr != nil {
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
