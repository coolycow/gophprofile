package service

import (
	"context"
	"fmt"

	"github.com/coolycow/gophprofile/internal/config"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/minio/minio-go/v7"
)

// HealthService интерфейс для проверки доступности сервисов
type HealthService interface {
	CheckDatabaseStatus(ctx context.Context) error
	CheckMinioStatus(ctx context.Context) error
	CheckRabbitMQStatus(ctx context.Context) error
	CheckWorkerStatus(ctx context.Context) error
}

// RabbitMQHealthConn минимальный контракт для проверки RabbitMQ в /health.
type RabbitMQHealthConn interface {
	IsClosed() bool
}

// healthService реализация HealthService
type healthService struct {
	repo        repository.GophProfileRepository
	cfg         *config.ConfigServer
	minioClient *minio.Client
	rabbitMQ    RabbitMQHealthConn
}

// NewHealthService инициализация HealthService. rabbitMQ может быть nil — проверка RabbitMQ в /health пропускается.
func NewHealthService(repo repository.GophProfileRepository, cfg *config.ConfigServer, minioClient *minio.Client, rabbitMQ RabbitMQHealthConn) HealthService {
	return &healthService{
		repo:        repo,
		cfg:         cfg,
		minioClient: minioClient,
		rabbitMQ:    rabbitMQ,
	}
}

// CheckDatabaseStatus проверяет доступность БД
func (s *healthService) CheckDatabaseStatus(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

// CheckMinioStatus проверяет доступность MinIO
func (s *healthService) CheckMinioStatus(ctx context.Context) error {
	exists, err := s.minioClient.BucketExists(ctx, s.cfg.MinioBucketName)

	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("bucket %s does not exist", s.cfg.MinioBucketName)
	}

	return nil
}

// CheckRabbitMQStatus проверяет доступность RabbitMQ
func (s *healthService) CheckRabbitMQStatus(ctx context.Context) error {
	_ = ctx
	if s.rabbitMQ == nil {
		return nil
	}
	if s.rabbitMQ.IsClosed() {
		return fmt.Errorf("connection closed")
	}
	return nil
}

// CheckWorkerStatus проверяет доступность воркера
func (s *healthService) CheckWorkerStatus(ctx context.Context) error {
	return nil
}
