package middleware

import (
	"net/http"
	"strings"

	"github.com/coolycow/gophprofile/internal/config"
	"github.com/gin-gonic/gin"
)

// CORS настраивает заголовки CORS. CORSAllowedOrigins: список origin через запятую или "*".
func CORS(cfg *config.ConfigServer) gin.HandlerFunc {
	if cfg == nil {
		return func(c *gin.Context) { c.Next() }
	}

	// Получаем список origin через запятую или "*"
	raw := strings.TrimSpace(cfg.CORSAllowedOrigins)
	if raw == "" {
		raw = "*"
	}

	// Разделяем список origin через запятую
	allowed := strings.Split(raw, ",")
	for i := range allowed {
		allowed[i] = strings.TrimSpace(allowed[i])
	}

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		outOrigin := ""

		if len(allowed) == 1 && allowed[0] == "*" {
			if origin != "" {
				outOrigin = origin
			} else {
				outOrigin = "*"
			}
		} else {
			for _, o := range allowed {
				if o != "" && o == origin {
					outOrigin = origin
					break
				}
			}
			if origin != "" && outOrigin == "" {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
		}

		if outOrigin != "" {
			c.Header("Access-Control-Allow-Origin", outOrigin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-User-ID, If-None-Match")
		c.Header("Access-Control-Expose-Headers", "ETag, Content-Length, Content-Type")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
