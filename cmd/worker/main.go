package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coolycow/gophprofile/internal/config"
	"github.com/coolycow/gophprofile/internal/logger"
	"github.com/coolycow/gophprofile/internal/minio"
	"github.com/coolycow/gophprofile/internal/observability"
	"github.com/coolycow/gophprofile/internal/rabbitmq"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/coolycow/gophprofile/internal/service"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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

	obsCfg := observability.FromWorker(cfg)
	if obsCfg.OtelServiceName == "" {
		obsCfg.OtelServiceName = "gophprofile-worker"
	}

	// Инициализируем логер
	if err = logger.Initialize(cfg.LogLevel, obsCfg.LogFormat); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	observability.InitMetrics()

	initCtx := context.Background()
	shutdownTracing, err := observability.InitTracing(initCtx, obsCfg)
	if err != nil {
		logger.Log.Error("Failed to initialize tracing", "error", err)
	}
	defer func() {
		if shutdownTracing == nil {
			return
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracing(shutdownCtx); err != nil {
			logger.Log.Error("Failed to shutdown tracing", "error", err)
		}
	}()

	metricsCtx, metricsCancel := context.WithCancel(context.Background())
	defer metricsCancel()

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

	if pgRepo, ok := repo.(*repository.PostgresRepository); ok {
		observability.StartDBStatsCollector(metricsCtx, pgRepo.DB(), 15*time.Second)
	}

	// Если флаг запуска миграций установлен, выполняем миграции
	if cfg.RunMigrations {
		if err = repo.RunMigrations(); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		logger.Log.Info("Run migrations succeeded")
	}

	// В конце работы приложения необходимо правильно закрыть хранилище.
	defer func() {
		if err = repo.Close(); err != nil {
			logger.Log.Error("Error closing repository", "error", err)
		}
	}()

	// Инициализируем MinIO клиент
	minioClient, err := minio.NewMinioClient(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucketName, cfg.MinioUseSSL)
	if err != nil {
		logger.Log.Error("Failed to initialize minio client", "error", err)
		os.Exit(1)
	}

	// Инициализируем сервис работы с аватарками
	workerService := service.NewWorkerService(repo, cfg, minioClient)

	metricsSrv, err := observability.StartMetricsServer(metricsCtx, obsCfg.MetricsAddr)
	if err != nil {
		logger.Log.Error("Failed to start metrics server", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = observability.ShutdownMetricsServer(shutdownCtx, metricsSrv)
	}()

	// Создаем контекст для завершения работы воркера
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	// Подключение к RabbitMQ серверу
	amqpURI := rabbitmq.BuildURI(cfg.RabbitMQUser, cfg.RabbitMQPassword, cfg.RabbitMQHost, cfg.RabbitMQPort, cfg.RabbitMQVHost)
	rabbitConn, err := amqp.Dial(amqpURI)
	if err != nil {
		logger.Log.Error("Failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer rabbitConn.Close()

	// Канал для потребления
	ch, err := rabbitConn.Channel()
	if err != nil {
		logger.Log.Error("Failed to open a channel", "error", err)
		os.Exit(1)
	}
	defer ch.Close()

	// Отдельный канал для republish (не смешиваем publish/consume на одном канале).
	pubCh, err := rabbitConn.Channel()
	if err != nil {
		logger.Log.Error("Failed to open publish channel", "error", err)
		os.Exit(1)
	}
	defer pubCh.Close()

	queue, err := rabbitmq.EnsureAvatarsTopology(ch)
	if err != nil {
		logger.Log.Error("Failed to declare RabbitMQ topology", "error", err)
		os.Exit(1)
	}
	logger.Log.Info("Declared RabbitMQ topology", "queue", queue.Name)

	// Регистрация потребителя
	consumerTag := fmt.Sprintf("%s-%d", rabbitmq.ConsumerTagDefault, os.Getpid())
	msgs, err := rabbitmq.ConsumeAvatarJobs(ch, 1, consumerTag)
	if err != nil {
		logger.Log.Error("Failed to register rabbit consumer", "error", err)
		os.Exit(1)
	}

	// Завершение работы потребителя
	go func() {
		<-ctx.Done()
		if cerr := ch.Cancel(consumerTag, false); cerr != nil {
			logger.Log.Warn("rabbitmq consumer cancel", "error", cerr)
		}
	}()

	runConsumer(ctx, msgs, workerService, pubCh)
}

// runConsumer обрабатывает доставки до отмены контекста или закрытия канала.
func runConsumer(ctx context.Context, msgs <-chan amqp.Delivery, workerService service.WorkerService, pubCh *amqp.Channel) {
	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("worker stopping (context canceled)")
			return
		case d, ok := <-msgs:
			if !ok {
				logger.Log.Info("rabbit consumer channel closed")
				return
			}
			if !processDelivery(ctx, d, workerService, pubCh) {
				return
			}
		}
	}
}

// processDelivery обрабатывает одно сообщение из очереди. false — прекратить цикл потребителя.
func processDelivery(ctx context.Context, d amqp.Delivery, workerService service.WorkerService, pubCh *amqp.Channel) bool {
	jobCtx := observability.ExtractAMQPContext(ctx, d.Headers)
	jobCtx, span := otel.Tracer(observability.Tracer()).Start(jobCtx, "rabbitmq.consume")
	defer span.End()
	span.SetAttributes(attribute.String("messaging.system", "rabbitmq"))

	job, decErr := rabbitmq.DecodeAvatarJob(d.Body)
	if decErr != nil {
		logger.Log.Warn("invalid job payload", "error", decErr, "body", string(d.Body))
		span.RecordError(decErr)
		span.SetStatus(codes.Error, decErr.Error())
		if nackErr := d.Nack(false, false); nackErr != nil {
			logger.Log.Error("rabbitmq nack failed", "error", nackErr)
			return false
		}
		return true
	}

	span.SetAttributes(attribute.String("job.type", string(job.Type)))

	jobCtx, jobCancel := context.WithTimeout(jobCtx, 5*time.Minute)
	defer jobCancel()

	logger.Log.Info("received avatar job",
		"type", string(job.Type),
		"avatar_id", job.AvatarID,
		"user_id", job.UserID,
		"s3_key", job.S3Key,
		"thumbnail_s3_keys", job.ThumbnailS3Keys,
		"retry", rabbitmq.HeaderRetryCount(d.Headers),
	)

	var err error
	switch job.Type {
	case rabbitmq.AvatarJobTypeProcess:
		err = workerService.ProcessAvatar(jobCtx, job.AvatarID)
	case rabbitmq.AvatarJobTypeDeleteByS3Key:
		err = workerService.DeleteAvatarByS3Key(jobCtx, job.S3Key, job.ThumbnailS3Keys)
	default:
		err = fmt.Errorf("unsupported job type: %s", job.Type)
	}

	if err != nil {
		logger.Log.Error("avatar job failed",
			"type", string(job.Type),
			"error", err,
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		retry := rabbitmq.HeaderRetryCount(d.Headers)
		if rabbitmq.IsTransientAvatarJobError(err) && retry < rabbitmq.MaxAvatarJobRetries {
			delay := rabbitmq.RetryBackoffDuration(retry)
			select {
			case <-ctx.Done():
				_ = d.Nack(false, true)
				return false
			case <-time.After(delay):
			}
			if rerr := rabbitmq.RepublishAvatarJob(jobCtx, pubCh, d.Body, job, retry+1); rerr != nil {
				logger.Log.Error("republish avatar job failed", "error", rerr)
				if nackErr := d.Nack(false, false); nackErr != nil {
					logger.Log.Error("rabbitmq nack failed", "error", nackErr)
					return false
				}
				return true
			}
			if ackErr := d.Ack(false); ackErr != nil {
				logger.Log.Error("rabbitmq ack failed", "error", ackErr)
				return false
			}
			return true
		}
		if nackErr := d.Nack(false, false); nackErr != nil {
			logger.Log.Error("rabbitmq nack failed", "error", nackErr)
			return false
		}
		return true
	}

	if err := d.Ack(false); err != nil {
		logger.Log.Error("rabbitmq ack failed", "error", err)
		return false
	}
	return true
}
