package service

import (
	"context"

	"github.com/coolycow/gophprofile/internal/config"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/minio/minio-go/v7"
)

// WorkerService операции воркера по сообщениям из очереди.
type WorkerService interface {
	ProcessAvatar(ctx context.Context, avatarID string) error
	DeleteAvatarByS3Key(ctx context.Context, s3Key string) error
}

// Реализация сервисного слоя
type workerService struct {
	repo        repository.GophProfileRepository
	cfg         *config.ConfigWorker
	minioClient *minio.Client
}

// NewWorkerService создаёт сервис воркера (асинхронная обработка после HTTP).
func NewWorkerService(repo repository.GophProfileRepository, cfg *config.ConfigWorker, minioClient *minio.Client) WorkerService {
	return &workerService{
		repo:        repo,
		cfg:         cfg,
		minioClient: minioClient,
	}
}

// ProcessAvatar обрабатывает аватарку
func (s *workerService) ProcessAvatar(ctx context.Context, avatarID string) error {
	_ = ctx
	_ = avatarID
	_ = s
	return nil
}

// DeleteAvatarByS3Key удаляет аватарку из S3 по ключу
func (s *workerService) DeleteAvatarByS3Key(ctx context.Context, s3Key string) error {
	err := s.minioClient.RemoveObject(ctx, s.cfg.MinioBucketName, s3Key, minio.RemoveObjectOptions{})

	if err != nil {
		return err
	}

	return nil
}
