package router

import (
	"net"
	"strings"

	"github.com/coolycow/gophprofile/internal/config"
	"github.com/coolycow/gophprofile/internal/handler"
	"github.com/coolycow/gophprofile/internal/middleware"
	"github.com/coolycow/gophprofile/internal/observer/audit"
	"github.com/coolycow/gophprofile/internal/repository"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

// parseTrustedSubnet парсит строку CIDR в структуру net.IPNet
func parseTrustedSubnet(cidr string) (*net.IPNet, error) {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return nil, nil
	}

	_, ipNet, err := net.ParseCIDR(cidr)
	return ipNet, err
}

// setupURLRoutes настраивает маршруты для URL-сервиса
func setupRoutes(
	r *gin.Engine,
	cfg *config.ConfigServer,
	repo repository.GophProfileRepository,
	auditNotifier *audit.Notifier,
	minioClient *minio.Client,
	avatarJobs service.AvatarJobPublisher,
	rabbitMQConn service.RabbitMQHealthConn,
) {
	// Инициализация сервисов
	avatarService := service.NewAvatarService(cfg, repo, minioClient, avatarJobs)
	healthService := service.NewHealthService(repo, cfg, minioClient, rabbitMQConn)
	userService := service.NewUserService(cfg, repo)

	// Проверка работоспособности сервиса (не требует идентификации пользователя)
	r.GET("/health", handler.HealthHandler(healthService))

	// Группа маршрутов для API
	api := r.Group("/api/v1")

	// Маршруты для авторизации (TODO: их нет в задании, но как-то же нужно создать пользователя)
	api.POST("/auth/register", handler.RegisterHandler(userService))
	api.POST("/auth/login", handler.LoginHandler(userService))
	api.POST("/auth/refresh", handler.RefreshHandler(userService))

	// Маршруты для получения аватара по ID и по ID пользователя (не требуют идентификации пользователя)
	api.GET("/avatars/:avatar_id", handler.GetAvatarByIDHandler(avatarService))
	api.GET("/avatars/:avatar_id/metadata", handler.GetAvatarMetadataHandler(avatarService))

	api.GET("/users/:user_id/avatar", handler.GetAvatarByUserIDHandler(avatarService))
	api.GET("/users/:user_id/avatars", handler.GetAvatarsByUserIDHandler(avatarService))

	// Группа маршрутов для защищённых маршрутов (идентификация пользователя через X-User-ID от шлюза)
	protected := api.Group("")
	protected.Use(middleware.RequireXUserID())
	protected.POST("/avatars", handler.PostAvatarHandler(avatarService, auditNotifier))
	protected.DELETE("/avatars/:avatar_id", handler.DeleteAvatarByIDHandler(avatarService))
	protected.DELETE("/users/:user_id/avatar", handler.DeleteAvatarByUserIDHandler(avatarService))
}
