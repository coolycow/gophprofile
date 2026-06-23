package service

import (
	"context"
	"fmt"

	"github.com/coolycow/gophprofile/internal/config"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/coolycow/gophprofile/internal/resilience"
	"github.com/minio/minio-go/v7"
	"github.com/sony/gobreaker"
)

const avatarJobsQueueName = "gophprofile.avatars"

// HealthService интерфейс для проверки доступности сервисов
type HealthService interface {
	CheckDatabaseStatus(ctx context.Context) error
	CheckMinioStatus(ctx context.Context) error
	CheckRabbitMQStatus(ctx context.Context) error
	CheckWorkerStatus(ctx context.Context) error
}

// RabbitMQHealthConn проверка RabbitMQ и очереди воркера для /health.
type RabbitMQHealthConn interface {
	IsClosed() bool
	QueueConsumerCount(queueName string) (int, error)
}

// healthService реализация HealthService
type healthService struct {
	repo         repository.GophProfileRepository
	cfg          *config.ConfigServer
	minioClient  *minio.Client
	rabbitMQ     RabbitMQHealthConn
	minioBreaker *gobreaker.CircuitBreaker
}

// NewHealthService инициализация HealthService. rabbitMQ может быть nil — проверки брокера/воркера пропускаются.
func NewHealthService(repo repository.GophProfileRepository, cfg *config.ConfigServer, minioClient *minio.Client, rabbitMQ RabbitMQHealthConn, minioBreaker *gobreaker.CircuitBreaker) HealthService {
	return &healthService{
		repo:         repo,
		cfg:          cfg,
		minioClient:  minioClient,
		rabbitMQ:     rabbitMQ,
		minioBreaker: minioBreaker,
	}
}

// CheckDatabaseStatus проверяет доступность БД
func (s *healthService) CheckDatabaseStatus(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

// CheckMinioStatus проверяет доступность MinIO
func (s *healthService) CheckMinioStatus(ctx context.Context) error {
	exists, err := resilience.Execute(s.minioBreaker, func() (bool, error) {
		return s.minioClient.BucketExists(ctx, s.cfg.MinioBucketName)
	})

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

// CheckWorkerStatus проверяет, что к очереди заданий подключён хотя бы один consumer (воркер).
func (s *healthService) CheckWorkerStatus(ctx context.Context) error {
	_ = ctx
	if s.rabbitMQ == nil {
		return nil
	}
	if s.rabbitMQ.IsClosed() {
		return fmt.Errorf("rabbitmq connection closed")
	}
	n, err := s.rabbitMQ.QueueConsumerCount(avatarJobsQueueName)
	if err != nil {
		return fmt.Errorf("queue %s: %w", avatarJobsQueueName, err)
	}
	if n < 1 {
		return fmt.Errorf("no consumers on queue %s", avatarJobsQueueName)
	}
	return nil
}
