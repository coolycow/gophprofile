package handler

import (
	"net/http"

	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// HealthHandler проверяет доступность сервиса
func HealthHandler(srv service.HealthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbStatus := "ok"
		minioStatus := "ok"
		rabbitmqStatus := "ok"
		workerStatus := "ok"

		dbError := srv.CheckDatabaseStatus(c.Request.Context())
		minioError := srv.CheckMinioStatus(c.Request.Context())
		rabbitmqError := srv.CheckRabbitMQStatus(c.Request.Context())
		workerError := srv.CheckWorkerStatus(c.Request.Context())

		if dbError != nil {
			dbStatus = dbError.Error()
		}

		if minioError != nil {
			minioStatus = minioError.Error()
		}

		if rabbitmqError != nil {
			rabbitmqStatus = rabbitmqError.Error()
		}

		if workerError != nil {
			workerStatus = workerError.Error()
		}

		c.JSON(http.StatusOK, gin.H{"database": dbStatus, "minio": minioStatus, "rabbitmq": rabbitmqStatus, "worker": workerStatus})
	}
}
