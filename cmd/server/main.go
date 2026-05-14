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
	"github.com/coolycow/gophprofile/internal/observer/audit"
	"github.com/coolycow/gophprofile/internal/rabbitmq"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/coolycow/gophprofile/internal/router"
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
	cfg, err := config.InitConfigServer()

	// Если ошибка, выводим сообщение и завершаем программу
	if err != nil {
		log.Fatalf("Failed to initialize configuration: %v", err)
	}

	// Инициализируем логер
	if err = logger.Initialize(cfg.LogLevel); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

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
			logger.Log.Error("Error closing repository", zap.Error(err))
		}
	}()

	// Инициализируем нотифайер аудита (файл и/или URL из конфига; если оба пустые — приёмников не будет)
	auditNotifier, err := audit.NewNotifier(cfg.AuditFile, cfg.AuditURL)
	if err != nil {
		logger.Log.Fatal("Failed to initialize audit notifier", zap.Error(err))
	}

	// Инициализируем MinIO клиент
	minioClient, err := minio.NewMinioClient(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucketName, cfg.MinioUseSSL)
	if err != nil {
		logger.Log.Fatal("Failed to initialize minio client", zap.Error(err))
	}

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

	// Объявление exchange, очереди и привязок (topic)
	queue, err := rabbitmq.EnsureAvatarsTopology(ch)
	if err != nil {
		logger.Log.Fatal("Failed to declare RabbitMQ topology", zap.Error(err))
	}
	logger.Log.Info("Declared RabbitMQ topology", zap.String("queue", queue.Name))

	// Создание издателя заданий
	avatarJobPublisher := rabbitmq.NewAvatarJobPublisher(ch)

	rabbitHealth := rabbitmq.NewHealthConn(rabbitConn)

	// Инициализируем роутер
	r := router.NewRouter(cfg, repo, auditNotifier, minioClient, avatarJobPublisher, rabbitHealth)

	// Получаем адрес сервера из настроек и запускаем сервер
	serverAddress := cfg.GetServerAddress()
	logger.Log.Info("Running server ", zap.String("address", serverAddress))

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
			logger.Log.Fatal("server error", zap.Error(serveErr))
		}
	}()

	// Ожидаем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-quit
	logger.Log.Info("shutdown signal received", zap.String("signal", sig.String()))

	// Создаем контекст для завершения работы сервера
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Параллельно завершаем HTTP и gRPC.
	var shutdownWg sync.WaitGroup
	shutdownWg.Add(2)

	go func() {
		defer shutdownWg.Done()
	}()

	go func() {
		defer shutdownWg.Done()
		if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.Log.Error("graceful shutdown failed", zap.Error(shutdownErr))
		}
	}()

	shutdownWg.Wait()
}
