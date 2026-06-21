package middleware

import (
	"strconv"
	"time"

	"github.com/coolycow/gophprofile/internal/logger"
	"github.com/coolycow/gophprofile/internal/observability"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
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
		status := c.Writer.Status()
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		statusLabel := strconv.Itoa(status)

		observability.HTTPRequestsTotal.WithLabelValues(c.Request.Method, route, statusLabel).Inc()
		observability.HTTPRequestDuration.WithLabelValues(c.Request.Method, route, statusLabel).Observe(duration.Seconds())

		attrs := []any{
			"uri", c.Request.RequestURI,
			"method", c.Request.Method,
			"params", c.Request.URL.Query().Encode(),
			"duration", duration.String(),
			"status", status,
			"size", c.Writer.Size(),
		}
		if sc := trace.SpanFromContext(c.Request.Context()).SpanContext(); sc.IsValid() {
			attrs = append(attrs, "trace_id", sc.TraceID().String())
		}

		// Логируем информацию о запросе и ответе
		logger.Log.Info("HTTP request", attrs...)
	}
}
