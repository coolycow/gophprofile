package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/coolycow/gophprofile/internal/config"
	"github.com/coolycow/gophprofile/internal/logger"
	"github.com/coolycow/gophprofile/internal/minio"
	"github.com/coolycow/gophprofile/internal/observability"
	"github.com/coolycow/gophprofile/internal/observer/audit"
	"github.com/coolycow/gophprofile/internal/rabbitmq"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/coolycow/gophprofile/internal/router"
	"github.com/coolycow/gophprofile/internal/service"
	amqp "github.com/rabbitmq/amqp091-go"
)

// @title           GophProfile API
// @version         1.0
// @description     Microservice for user avatar management.
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey UserID
// @in header
// @name X-User-ID

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
	cfg, err := config.InitConfigServer()

	// Если ошибка, выводим сообщение и завершаем программу
	if err != nil {
		log.Fatalf("Failed to initialize configuration: %v", err)
	}

	obsCfg := observability.FromServer(cfg)
	if obsCfg.OtelServiceName == "" {
		obsCfg.OtelServiceName = "gophprofile-server"
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
	cfg.PrintServerConfig()

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

	// Инициализируем нотифайер аудита (файл и/или URL из конфига; если оба пустые — приёмников не будет)
	auditNotifier, err := audit.NewNotifier(cfg.AuditFile, cfg.AuditURL)
	if err != nil {
		logger.Log.Error("Failed to initialize audit notifier", "error", err)
		os.Exit(1)
	}

	// Инициализируем MinIO клиент
	minioClient, err := minio.NewMinioClient(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucketName, cfg.MinioUseSSL)
	if err != nil {
		logger.Log.Error("Failed to initialize minio client", "error", err)
		os.Exit(1)
	}

	// Подключение к RabbitMQ серверу
	amqpURI := rabbitmq.BuildURI(cfg.RabbitMQUser, cfg.RabbitMQPassword, cfg.RabbitMQHost, cfg.RabbitMQPort, cfg.RabbitMQVHost)
	rabbitConn, err := amqp.Dial(amqpURI)
	if err != nil {
		logger.Log.Error("Failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}

	// Создание канала для работы с RabbitMQ
	ch, err := rabbitConn.Channel()
	if err != nil {
		_ = rabbitConn.Close()
		logger.Log.Error("Failed to open a channel", "error", err)
		os.Exit(1)
	}

	// Объявление exchange, очереди и привязок (topic)
	queue, err := rabbitmq.EnsureAvatarsTopology(ch)
	if err != nil {
		logger.Log.Error("Failed to declare RabbitMQ topology", "error", err)
		os.Exit(1)
	}
	logger.Log.Info("Declared RabbitMQ topology", "queue", queue.Name)

	// Создание издателя заданий
	avatarJobPublisher := rabbitmq.NewAvatarJobPublisher(ch)

	rabbitHealth := rabbitmq.NewHealthConn(rabbitConn)

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

	// Инициализируем роутер
	r := router.NewRouter(cfg, repo, auditNotifier, minioClient, avatarJobPublisher, rabbitHealth)

	// Получаем адрес сервера из настроек и запускаем сервер
	serverAddress := cfg.GetServerAddress()
	logger.Log.Info("Running server ", "address", serverAddress)

	// Инициализируем сервер
	srv := &http.Server{
		Addr:    serverAddress,
		Handler: r,
	}

	// Запускаем сервер
	go func() {
		var serveErr error
		if cfg.EnableHTTPS {
			serveErr = srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
		} else {
			serveErr = srv.ListenAndServe()
		}
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Log.Error("server error", "error", serveErr)
			os.Exit(1)
		}
	}()

	// Ожидаем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-quit
	logger.Log.Info("shutdown signal received", "signal", sig.String())
	metricsCancel()

	// Создаем контекст для завершения работы сервера
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var shutdownWg sync.WaitGroup
	shutdownWg.Add(1)

	go func() {
		defer shutdownWg.Done()
		if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.Log.Error("graceful shutdown failed", "error", shutdownErr)
		}
	}()

	shutdownWg.Wait()

	if err := ch.Close(); err != nil {
		logger.Log.Warn("Failed to close RabbitMQ channel", "error", err)
	}
	if err := rabbitConn.Close(); err != nil {
		logger.Log.Warn("Failed to close RabbitMQ connection", "error", err)
	}
}
