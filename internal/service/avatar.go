package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/coolycow/gophprofile/internal/config"
	profileError "github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/model"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/go-playground/validator/v10"
	"github.com/minio/minio-go/v7"
)

// AvatarService Сервис для работы с аватарами
type AvatarService interface {
	UploadAvatar(ctx context.Context, userID string, file *multipart.File, fileHeader *multipart.FileHeader) (*model.Avatar, error)

	GetAvatarByID(ctx context.Context, avatarID string) (*model.Avatar, error)
	GetAvatarByUserID(ctx context.Context, userID string) (*model.Avatar, error)

	DeleteAvatarByID(ctx context.Context, avatarID string) error
	DeleteAvatarByUserID(ctx context.Context, userID string) error

	GetAvatarMetadataByID(ctx context.Context, avatarID string) (*model.AvatarMetadata, error)
	GetAvatarMetadataByUserID(ctx context.Context, userID string) (*model.AvatarMetadata, error)

	GetUserAvatars(ctx context.Context, userID string) ([]*model.Avatar, error)
}

// Реализация сервисного слоя
type avatarService struct {
	repo        repository.GophProfileRepository
	cfg         *config.ConfigServer
	validator   *validator.Validate
	minioClient *minio.Client
}

// NewAvatarService инициализация сервиса
func NewAvatarService(cfg *config.ConfigServer, repo repository.GophProfileRepository, minioClient *minio.Client) AvatarService {
	return &avatarService{
		repo:        repo,
		cfg:         cfg,
		validator:   validator.New(),
		minioClient: minioClient,
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

	// Валидация MIME типа (изображение, форматы: jpeg, png, webp)
	if err := s.validator.Var(fileHeader.Header.Get("Content-Type"), "required,mime:image/jpeg,image/png,image/webp"); err != nil {
		return nil, profileError.CustomError{
			Message:    fmt.Sprintf("mime type validation failed: %s", err),
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
	contentType := fileHeader.Header.Get("Content-Type")
	_, err = s.minioClient.PutObject(ctx, bucketName, objectName, bytes.NewReader(reader), fileHeader.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, profileError.CustomError{
			Message:    fmt.Sprintf("failed to upload file to MinIO: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	return s.repo.UploadAvatar(ctx, userID, fileHeader.Filename, contentType, fileHeader.Size, objectName, "[]", "completed", "pending")
}

// GetAvatarByID получает аватарку по ID
func (s *avatarService) GetAvatarByID(ctx context.Context, avatarID string) (*model.Avatar, error) {
	return s.repo.GetAvatarByID(ctx, avatarID)
}

// GetAvatarByUserID получает аватарку по ID пользователя
func (s *avatarService) GetAvatarByUserID(ctx context.Context, userID string) (*model.Avatar, error) {
	return s.repo.GetAvatarByUserID(ctx, userID)
}

// DeleteAvatarByID удаляет аватарку по ID
func (s *avatarService) DeleteAvatarByID(ctx context.Context, avatarID string) error {
	return s.repo.DeleteAvatarByID(ctx, avatarID)
}

// DeleteAvatarByUserID удаляет аватарку по ID пользователя
func (s *avatarService) DeleteAvatarByUserID(ctx context.Context, userID string) error {
	return s.repo.DeleteAvatarByUserID(ctx, userID)
}

// GetAvatarMetadataByID получает метаданные аватарки по ID
func (s *avatarService) GetAvatarMetadataByID(ctx context.Context, avatarID string) (*model.AvatarMetadata, error) {
	return s.repo.GetAvatarMetadataByID(ctx, avatarID)
}

// GetAvatarMetadataByUserID получает метаданные аватарки по ID пользователя
func (s *avatarService) GetAvatarMetadataByUserID(ctx context.Context, userID string) (*model.AvatarMetadata, error) {
	return s.repo.GetAvatarMetadataByUserID(ctx, userID)
}

// GetUserAvatars получает список аватарок пользователя
func (s *avatarService) GetUserAvatars(ctx context.Context, userID string) ([]*model.Avatar, error) {
	return s.repo.GetUserAvatars(ctx, userID)
}
