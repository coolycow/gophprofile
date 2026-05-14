package middleware

import (
	"net/http"
	"strings"
	"sync"

	"github.com/coolycow/gophprofile/internal/config"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimit ограничивает число запросов с одного IP (token bucket). /health не ограничиваем.
func RateLimit(cfg *config.ConfigServer) gin.HandlerFunc {
	if cfg == nil || !cfg.RateLimitEnabled {
		return func(c *gin.Context) { c.Next() }
	}

	// Получаем число запросов в секунду
	rps := cfg.RateLimitRPS
	if rps <= 0 {
		rps = 30
	}

	// Получаем размер «ведра» burst для rate limit
	burst := cfg.RateLimitBurst
	if burst < 1 {
		burst = 60
	}

	// Создаем map для хранения лимитеров по IP
	var mu sync.Mutex
	byIP := make(map[string]*rate.Limiter)

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/health" || strings.HasPrefix(path, "/health/") {
			c.Next()
			return
		}

		ip := strings.TrimSpace(c.ClientIP())
		if ip == "" {
			ip = "unknown"
		}

		mu.Lock()
		lim, ok := byIP[ip]
		if !ok {
			lim = rate.NewLimiter(rate.Limit(rps), burst)
			byIP[ip] = lim
		}
		mu.Unlock()

		if !lim.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
