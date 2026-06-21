package handler

import (
	"net/http"

	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// LiveHealthHandler отвечает 200, если HTTP-процесс жив (liveness probe).
func LiveHealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive"})
	}
}

// ReadyHealthHandler проверяет готовность принимать трафик (readiness probe).
// @Summary Readiness check
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /health/ready [get]
func ReadyHealthHandler(srv service.HealthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		writeHealthResponse(c, srv)
	}
}

// HealthHandler сохранён для обратной совместимости (= readiness).
func HealthHandler(srv service.HealthService) gin.HandlerFunc {
	return ReadyHealthHandler(srv)
}

func writeHealthResponse(c *gin.Context, srv service.HealthService) {
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

	status := http.StatusOK
	if dbError != nil || minioError != nil || rabbitmqError != nil || workerError != nil {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"database":  dbStatus,
		"minio":     minioStatus,
		"rabbitmq":  rabbitmqStatus,
		"worker":    workerStatus,
	})
}
