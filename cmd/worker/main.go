package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/coolycow/gophprofile/internal/config"
	"github.com/coolycow/gophprofile/internal/logger"
	"github.com/coolycow/gophprofile/internal/minio"
	"github.com/coolycow/gophprofile/internal/rabbitmq"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/coolycow/gophprofile/internal/service"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	// Вывод информации о сборке при старте
	fmt.Fprintf(os.Stdout, "Build version: %s\n", service.OrNA(buildVersion))
	fmt.Fprintf(os.Stdout, "Build date: %s\n", service.OrNA(buildDate))
	fmt.Fprintf(os.Stdout, "Build commit: %s\n", service.OrNA(buildCommit))

	// Инициализируем настройки (приоритет: окружение, флаги, дефолт)
	cfg, err := config.InitConfigWorker()

	// Если ошибка, выводим сообщение и завершаем программу
	if err != nil {
		log.Fatalf("Failed to initialize configuration: %v", err)
	}

	// Инициализируем логер
	if err = logger.Initialize(cfg.LogLevel); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Выводим настройки в лог
	cfg.PrintWorkerConfig()

	// Инициализируем репозиторий
	if cfg.DatabaseDSN == "" {
		log.Fatalf("Database DSN is required")
	}

	var repo repository.GophProfileRepository
	repo, err = repository.NewPostgresRepository(cfg.DatabaseDSN)

	if err != nil {
		log.Fatalf("Failed to initialize postgres repository: %v", err)
	}

	logger.Log.Info("Initialized postgres repository successfully")

	// Если флаг запуска миграций установлен, выполняем миграции
	if cfg.RunMigrations {
		if err = repo.RunMigrations(); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		logger.Log.Info("Run migrations succeeded")
		return
	}

	// В конце работы приложения необходимо правильно закрыть хранилище.
	defer func() {
		if err = repo.Close(); err != nil {
			logger.Log.Error("Error closing repository", zap.Error(err))
		}
	}()

	// Инициализируем MinIO клиент
	minioClient, err := minio.NewMinioClient(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucketName, cfg.MinioUseSSL)
	if err != nil {
		logger.Log.Fatal("Failed to initialize minio client", zap.Error(err))
	}

	// Инициализируем сервис работы с аватарками
	workerService := service.NewWorkerService(repo, cfg, minioClient, nil)

	// Создаем контекст для завершения работы воркера
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	// Подключение к RabbitMQ серверу
	amqpURI := rabbitmq.BuildURI(cfg.RabbitMQUser, cfg.RabbitMQPassword, cfg.RabbitMQHost, cfg.RabbitMQPort, cfg.RabbitMQVHost)
	rabbitConn, err := amqp.Dial(amqpURI)
	if err != nil {
		logger.Log.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	defer rabbitConn.Close()

	// Создание канала для работы с RabbitMQ
	ch, err := rabbitConn.Channel()
	if err != nil {
		logger.Log.Fatal("Failed to open a channel", zap.Error(err))
	}
	defer ch.Close()

	// Объявление очереди
	queue, err := rabbitmq.EnsureAvatarJobsQueue(ch)
	if err != nil {
		logger.Log.Fatal("Failed to declare avatar jobs queue", zap.Error(err))
	}
	logger.Log.Info("Declared queue successfully", zap.String("queue", queue.Name))

	// Регистрация потребителя
	consumerTag := fmt.Sprintf("%s-%d", rabbitmq.ConsumerTagDefault, os.Getpid())
	msgs, err := rabbitmq.ConsumeAvatarJobs(ch, 1, consumerTag)
	if err != nil {
		logger.Log.Fatal("Failed to register rabbit consumer", zap.Error(err))
	}

	// Завершение работы потребителя
	go func() {
		<-ctx.Done()
		if cerr := ch.Cancel(consumerTag, false); cerr != nil {
			logger.Log.Warn("rabbitmq consumer cancel", zap.Error(cerr))
		}
	}()

	// Запуск потребителя
	runConsumer(ctx, msgs, workerService)
}

// runConsumer обрабатывает доставки до отмены контекста или закрытия канала.
func runConsumer(ctx context.Context, msgs <-chan amqp.Delivery, workerService service.WorkerService) {
	for {
		select {
		// Завершение работы потребителя
		case <-ctx.Done():
			logger.Log.Info("worker stopping (context canceled)")
			return
		// Получение задания
		case d, ok := <-msgs:
			if !ok {
				logger.Log.Info("rabbit consumer channel closed")
				return
			}

			// Десериализация задания
			var job rabbitmq.AvatarProcessingJob
			if err := json.Unmarshal(d.Body, &job); err != nil {
				logger.Log.Warn("invalid job payload", zap.Error(err))
				if err := d.Nack(false, false); err != nil {
					logger.Log.Error("rabbitmq nack failed", zap.Error(err))
					return
				}
				continue
			}

			logger.Log.Info("received avatar processing job", zap.String("avatar_id", job.AvatarID))

			// TODO: обработка изображения, результат в S3 и обновление статуса в БД.
			if err := workerService.ProcessAvatar(ctx, job.AvatarID); err != nil {
				logger.Log.Error("failed to process avatar", zap.Error(err))
				if err := d.Nack(false, false); err != nil {
					logger.Log.Error("rabbitmq nack failed", zap.Error(err))
					return
				}
			}

			// Подтверждение обработки задания
			if err := d.Ack(false); err != nil {
				logger.Log.Error("rabbitmq ack failed", zap.Error(err))
				return
			}
		}
	}
}
