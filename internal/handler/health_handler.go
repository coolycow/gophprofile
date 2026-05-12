package handler

import (
	"net/http"

	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// HealthHandler проверяет доступность сервиса.
func HealthHandler(srv service.HealthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbStatus := "ok"
		minioStatus := "ok"
		rabbitmqStatus := "ok"
		workerStatus := "ok"

		// Проверяем доступность БД
		dbError := srv.CheckDatabaseStatus(c.Request.Context())
		// Проверяем доступность MinIO
		minioError := srv.CheckMinioStatus(c.Request.Context())
		// Проверяем доступность RabbitMQ
		rabbitmqError := srv.CheckRabbitMQStatus(c.Request.Context())
		// Проверяем доступность воркера
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

		// Если есть ошибки, то возвращаем 503 Service Unavailable
		status := http.StatusOK
		if dbError != nil || minioError != nil || rabbitmqError != nil || workerError != nil {
			status = http.StatusServiceUnavailable
		}

		// Возвращаем статус и информацию о доступности сервисов
		c.JSON(status, gin.H{"database": dbStatus, "minio": minioStatus, "rabbitmq": rabbitmqStatus, "worker": workerStatus})
	}
}
