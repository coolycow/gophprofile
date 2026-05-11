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
	"github.com/coolycow/gophprofile/internal/observer/audit"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/coolycow/gophprofile/internal/router"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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
		return
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
	minioClient, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		logger.Log.Fatal("Failed to initialize minio client", zap.Error(err))
	}

	logger.Log.Info("Initialized minio client successfully")

	// Создаем бакет если он не существует
	err = minioClient.MakeBucket(context.Background(), cfg.MinioBucketName, minio.MakeBucketOptions{})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(context.Background(), cfg.MinioBucketName)
		if errBucketExists == nil && exists {
			logger.Log.Info("We already own %s", zap.String("bucket", cfg.MinioBucketName))
		} else {
			logger.Log.Fatal("Failed to create bucket", zap.Error(err))
		}
	}

	logger.Log.Info("Created bucket successfully")

	// Инициализируем роутер
	r := router.NewRouter(cfg, repo, auditNotifier, minioClient)

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
