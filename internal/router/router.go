// Package router настраивает маршруты и middleware HTTP-сервера.
package router

import (
	"github.com/coolycow/gophprofile/internal/config"
	"github.com/coolycow/gophprofile/internal/middleware"
	"github.com/coolycow/gophprofile/internal/observer/audit"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

// NewRouter создаёт HTTP-роутер с маршрутами сервиса коротких ссылок, gzip, логированием и pprof.
func NewRouter(
	cfg *config.ConfigServer,
	repo repository.GophProfileRepository,
	auditNotifier *audit.Notifier,
	minioClient *minio.Client,
	avatarJobs service.AvatarJobPublisher,
	rabbitMQConn service.RabbitMQHealthConn,
) *gin.Engine {
	router := gin.Default()

	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.RequestLogger())
	router.Use(middleware.ErrorHandler())
	router.Use(middleware.RequestGzip())

	setupRoutes(router, cfg, repo, auditNotifier, minioClient, avatarJobs, rabbitMQConn)

	return router
}
