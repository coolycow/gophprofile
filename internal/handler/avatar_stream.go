package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/coolycow/gophprofile/internal/model"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// normalizeETagCompare нормализует ETag для сравнения
func normalizeETagCompare(etag string) string {
	etag = strings.TrimSpace(etag)
	return strings.Trim(etag, `"`)
}

// streamAvatarResponse отдаёт аватар с учётом ?size=&format=, Cache-Control и ETag (304 при совпадении).
func streamAvatarResponse(c *gin.Context, srv service.AvatarService, avatar *model.Avatar) {
	size := c.Query("size")
	format := c.Query("format")

	// Подготавливаем аватар для загрузки
	dl, err := srv.PrepareAvatarDownload(c.Request.Context(), avatar, size, format)
	if err != nil {
		pushServiceError(c, err)
		return
	}
	defer dl.Body.Close()

	// Устанавливаем заголовок Cache-Control
	c.Header("Cache-Control", "max-age=86400")
	if dl.ETag != "" {
		c.Header("ETag", dl.ETag)
	}

	// Проверяем, если клиент отправил If-None-Match и ETag совпадает, то возвращаем 304 Not Modified
	inm := strings.TrimSpace(c.GetHeader("If-None-Match"))
	if inm != "" && dl.ETag != "" && normalizeETagCompare(inm) == normalizeETagCompare(dl.ETag) {
		c.Status(http.StatusNotModified)
		return
	}

	// Устанавливаем заголовок Content-Type
	c.Header("Content-Type", dl.ContentType)
	if dl.ContentLength >= 0 {
		c.Header("Content-Length", strconv.FormatInt(dl.ContentLength, 10))
	}

	// Отправляем аватарку клиенту
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, dl.Body); err != nil {
		return
	}
}
