package middleware

import (
	"time"

	"github.com/coolycow/gophprofile/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLogger — middleware-логер для входящих HTTP-запросов.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Фиксируем время начала обработки запроса
		start := time.Now()

		// Передаём управление другим Middleware или Handler
		c.Next()

		// Вычисляем длительность обработки запроса
		duration := time.Since(start)

		// Логируем информацию о запросе и ответе
		logger.Log.Info("HTTP request",
			zap.String("uri", c.Request.RequestURI),
			zap.String("method", c.Request.Method),
			zap.Duration("duration", duration),
			zap.Int("status", c.Writer.Status()),
			zap.Int("size", c.Writer.Size()),
		)
	}
}
