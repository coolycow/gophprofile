package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TrustedSubnetInternalStats ограничивает доступ к GET /api/internal/stats по X-Real-IP и trusted_subnet.
//
// Если trusted == nil (в конфиге не задана доверенная подсеть), отвечает 403 на любой запрос:
// эндпоинт считается выключенным, пока администратор явно не укажет CIDR. Иначе при пустой
// подсети статистика оказалась бы доступна любому клиенту, что для internal API нежелательно.
func TrustedSubnetInternalStats(trusted *net.IPNet) gin.HandlerFunc {
	return func(c *gin.Context) {
		if trusted == nil {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		clientIP := strings.TrimSpace(c.GetHeader("X-Real-IP"))
		ip := net.ParseIP(clientIP)
		if ip == nil || !trusted.Contains(ip) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}
