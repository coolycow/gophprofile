package router

import (
	"net"
	"strings"

	"github.com/coolycow/gophprofile/internal/config"
	"github.com/coolycow/gophprofile/internal/handler"
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
) {
	srv := service.NewAvatarService(cfg, repo, minioClient)

	r.GET("/health", handler.HealthHandler(srv))

	// Группа маршрутов с опциональной аутентификацией
	authGroup := r.Group("/")

	authGroup.POST("/api/v1/avatars", handler.PostAvatarHandler(srv, auditNotifier))

	// Маршруты для получения аватара по ID и по ID пользователя
	authGroup.GET("/api/v1/avatars/:avatar_id", handler.GetAvatarByIDHandler(srv))
	authGroup.GET("/api/v1/users/:user_id/avatar", handler.GetAvatarByUserIDHandler(srv))

	// Маршруты для удаления аватара по ID и по ID пользователя
	authGroup.DELETE("/api/v1/avatars/:avatar_id", handler.DeleteAvatarByIDHandler(srv))
	authGroup.DELETE("/api/v1/users/:user_id/avatar", handler.DeleteAvatarByUserIDHandler(srv))

	// Получение метаданных аватарки
	authGroup.GET("/api/v1/avatars/:avatar_id/metadata", handler.GetAvatarMetadataHandler(srv))

	// Список аватарок пользователя
	authGroup.GET("/api/v1/users/:user_id/avatars", handler.GetUserAvatarsHandler(srv))
}
