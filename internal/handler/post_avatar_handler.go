package handler

import (
	"mime/multipart"
	"net/http"
	"time"

	apperror "github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/model"
	"github.com/coolycow/gophprofile/internal/observer/audit"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// pickMultipartImageFile поле файла: приоритет `file` (как в ТЗ), затем `image` (как в исходном коде клиента).
func pickMultipartImageFile(c *gin.Context) (*multipart.FileHeader, error) {
	if fh, err := c.FormFile("file"); err == nil {
		return fh, nil
	}

	return c.FormFile("image")
}

// PostAvatarHandler загружает аватарку
func PostAvatarHandler(srv service.AvatarService, _ *audit.Notifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		// Получаем файл из запроса
		fileHeader, err := pickMultipartImageFile(c)
		if err != nil {
			pushServiceError(c, apperror.CustomError{
				Message:    err.Error(),
				StatusCode: http.StatusBadRequest,
			})
			return
		}

		// Открываем файл
		rc, err := fileHeader.Open()
		if err != nil {
			pushServiceError(c, err)
			return
		}
		defer rc.Close()

		// Загружаем аватарку
		avatar, err := srv.UploadAvatar(c.Request.Context(), userID, rc, fileHeader)
		if err != nil {
			pushServiceError(c, err)
			return
		}

		// Получаем время создания аватарки
		created := time.Now().UTC()
		if avatar.CreatedAt != nil {
			created = avatar.CreatedAt.UTC()
		}

		// Формируем ответ
		resp := model.AvatarUploadResponse{
			ID:        avatar.ID,
			UserID:    avatar.UserID,
			URL:       PublicAPIBaseURL(c) + "/api/v1/avatars/" + avatar.ID,
			Status:    "processing",
			CreatedAt: created,
		}

		// Возвращаем ответ
		c.JSON(http.StatusCreated, resp)
	}
}
