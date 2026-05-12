package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/model"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
	minio "github.com/minio/minio-go/v7"
)

// streamAvatarFromStorage отдаёт тело объекта из MinIO по записи avatar (S3Key — ключ в бакете, не путь на диске).
func streamAvatarFromStorage(c *gin.Context, srv service.AvatarService, avatar *model.Avatar) {
	// Открываем объект аватарки из MinIO
	obj, err := srv.OpenAvatarObject(c.Request.Context(), avatar)
	if err != nil {
		_ = c.Error(error.CustomError{
			Message:    err.Error(),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}
	defer obj.Close()

	// Получаем статистику объекта аватарки из MinIO
	stat, err := obj.Stat()
	if err != nil {
		// Если объект не найден, возвращаем 404
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			_ = c.Error(error.CustomError{
				Message:    "Avatar file not found in storage",
				StatusCode: http.StatusNotFound,
			})
			return
		}

		// Если ошибка, возвращаем 500
		_ = c.Error(error.CustomError{
			Message:    err.Error(),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// Получаем MIME тип аватарки из MinIO
	ct := strings.TrimSpace(avatar.MimeType)

	// Если MIME тип не найден, устанавливаем из статистики объекта аватарки из MinIO
	if stat.ContentType != "" {
		ct = stat.ContentType
	}

	// Если MIME тип не найден, устанавливаем default
	if ct == "" {
		ct = "application/octet-stream"
	}

	// Устанавливаем заголовки для ответа
	c.Header("Content-Type", ct)

	// Устанавливаем Content-Length
	if stat.Size >= 0 {
		c.Header("Content-Length", strconv.FormatInt(stat.Size, 10))
	}

	// Устанавливаем статус ответа
	c.Status(http.StatusOK)

	// Копируем тело объекта аватарки в ответ
	if _, err := io.Copy(c.Writer, obj); err != nil {
		return
	}
}
