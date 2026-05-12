package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// PublicAPIBaseURL схема и хост для публичных ссылок на API (учёт X-Forwarded-* за прокси).
func PublicAPIBaseURL(c *gin.Context) string {
	scheme := "http"

	// Если запрос TLS, то устанавливаем HTTPS
	if c.Request.TLS != nil {
		scheme = "https"
	}

	// Получаем протокол из заголовка X-Forwarded-Proto
	if xf := strings.TrimSpace(c.Request.Header.Get("X-Forwarded-Proto")); xf == "https" || xf == "http" {
		scheme = xf
	}

	// Получаем хост из заголовка X-Forwarded-Host
	host := strings.TrimSpace(c.Request.Host)
	if xh := strings.TrimSpace(c.Request.Header.Get("X-Forwarded-Host")); xh != "" {
		host = xh
	}

	// Если хост пустой, то устанавливаем localhost
	if host == "" {
		host = "localhost"
	}

	// Возвращаем URL
	return scheme + "://" + host
}
