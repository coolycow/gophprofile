package service

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

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

	DeleteAvatarByID(ctx context.Context, callerUserID, avatarID string) error
	DeleteAvatarByUserID(ctx context.Context, callerUserID, pathUserID string) error

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

	return s.repo.UploadAvatar(ctx, userID, fileHeader.Filename, contentType, info.Size, info.Key, "[]", "completed", "pending")
}

// GetAvatarByID получает аватарку по ID
func (s *avatarService) GetAvatarByID(ctx context.Context, avatarID string) (*model.Avatar, error) {
	return s.repo.GetAvatarByID(ctx, avatarID)
}

// GetAvatarByUserID получает аватарку по ID пользователя
func (s *avatarService) GetAvatarByUserID(ctx context.Context, userID string) (*model.Avatar, error) {
	return s.repo.GetAvatarByUserID(ctx, userID)
}

// DeleteAvatarByID удаляет аватарку по ID, если callerUserID совпадает с владельцем.
func (s *avatarService) DeleteAvatarByID(ctx context.Context, callerUserID, avatarID string) error {
	callerUserID = strings.TrimSpace(callerUserID)
	avatarID = strings.TrimSpace(avatarID)

	a, err := s.repo.GetAvatarByID(ctx, avatarID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return profileError.CustomError{
				Message:    "avatar not found",
				StatusCode: http.StatusNotFound,
			}
		}
		return err
	}

	if !strings.EqualFold(strings.TrimSpace(a.UserID), callerUserID) {
		return profileError.CustomError{
			Message:    "Forbidden",
			Details:    "You can only delete your own avatars",
			StatusCode: http.StatusForbidden,
		}
	}

	return s.repo.DeleteAvatarByID(ctx, avatarID)
}

// DeleteAvatarByUserID удаляет аватар пользователя; pathUserID должен совпадать с callerUserID.
func (s *avatarService) DeleteAvatarByUserID(ctx context.Context, callerUserID, pathUserID string) error {
	callerUserID = strings.TrimSpace(callerUserID)
	pathUserID = strings.TrimSpace(pathUserID)

	if !strings.EqualFold(callerUserID, pathUserID) {
		return profileError.CustomError{
			Message:    "Forbidden",
			Details:    "You can only delete your own avatars",
			StatusCode: http.StatusForbidden,
		}
	}

	_, err := s.repo.GetAvatarByUserID(ctx, pathUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return profileError.CustomError{
				Message:    "avatar not found",
				StatusCode: http.StatusNotFound,
			}
		}
		return err
	}

	return s.repo.DeleteAvatarByUserID(ctx, pathUserID)
}

// GetUserAvatars получает список аватарок пользователя
func (s *avatarService) GetUserAvatars(ctx context.Context, userID string) ([]*model.Avatar, error) {
	return s.repo.GetUserAvatars(ctx, userID)
}
