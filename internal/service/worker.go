package service

import (
	"context"

	"github.com/coolycow/gophprofile/internal/config"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/minio/minio-go/v7"
	amqp "github.com/rabbitmq/amqp091-go"
)

// WorkerService интерфейс для работы с аватарками воркером
type WorkerService interface {
	ProcessAvatar(ctx context.Context, avatarID string) error
	DeleteAvatar(ctx context.Context, avatarID string) error
}

// workerService реализация WorkerService для воркера
type workerService struct {
	repo        repository.GophProfileRepository
	cfg         *config.ConfigWorker
	minioClient *minio.Client
	rabbitMQ    *amqp.Channel
}

// NewWorkerService инициализация WorkerService для воркера
func NewWorkerService(repo repository.GophProfileRepository, cfg *config.ConfigWorker, minioClient *minio.Client, rabbitMQ *amqp.Channel) WorkerService {
	return &workerService{
		repo:        repo,
		cfg:         cfg,
		minioClient: minioClient,
		rabbitMQ:    rabbitMQ,
	}
}

// ProcessAvatar обрабатывает аватарку
func (s *workerService) ProcessAvatar(ctx context.Context, avatarID string) error {
	return nil
}

// DeleteAvatar удаляет аватарку
func (s *workerService) DeleteAvatar(ctx context.Context, avatarID string) error {
	return nil
}
