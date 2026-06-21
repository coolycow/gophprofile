// Package router настраивает маршруты и middleware HTTP-сервера.
package router

import (
	"os"
	"strings"

	"github.com/coolycow/gophprofile/internal/config"
	"github.com/coolycow/gophprofile/internal/middleware"
	"github.com/coolycow/gophprofile/internal/observer/audit"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// setGinModeFromEnvOrConfig устанавливает режим Gin из окружения или конфига
func setGinModeFromEnvOrConfig(cfg *config.ConfigServer) {
	env := strings.TrimSpace(os.Getenv("GIN_MODE"))
	switch env {
	case gin.DebugMode, gin.ReleaseMode, gin.TestMode:
		gin.SetMode(env)
	case "":
		// без GIN_MODE — release, кроме явного LOG_LEVEL=debug
		if cfg != nil && strings.EqualFold(strings.TrimSpace(cfg.LogLevel), "debug") {
			gin.SetMode(gin.DebugMode)
			return
		}
		gin.SetMode(gin.ReleaseMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}
}

// NewRouter создаёт HTTP-роутер с маршрутами сервиса коротких ссылок, gzip, логированием и pprof.
func NewRouter(
	cfg *config.ConfigServer,
	repo repository.GophProfileRepository,
	auditNotifier *audit.Notifier,
	minioClient *minio.Client,
	avatarJobs service.AvatarJobPublisher,
	rabbitMQConn service.RabbitMQHealthConn,
) *gin.Engine {
	setGinModeFromEnvOrConfig(cfg)

	// gin.Default() даёт встроенный Logger + Recovery и шумит в лог; свой лог — RequestLogger.
	router := gin.New()
	router.Use(gin.Recovery())

	router.Use(otelgin.Middleware("gophprofile-server"))

	router.Use(middleware.CORS(cfg))
	router.Use(middleware.RateLimit(cfg))
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.RequestLogger())
	router.Use(middleware.ErrorHandler())
	router.Use(middleware.RequestGzip())

	setupRoutes(router, cfg, repo, auditNotifier, minioClient, avatarJobs, rabbitMQConn)

	return router
}
