package handler

import (
	"net/http"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/observer/audit"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// PostAvatarHandler загружает аватарку
func PostAvatarHandler(srv service.AvatarService, auditNotifier *audit.Notifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем ID пользователя из контекста
		userID := c.GetString("user_id")

		// Получаем файл из запроса
		file, fileHeader, err := c.Request.FormFile("file")

		// Если ошибка, возвращаем 400
		if err != nil {
			_ = c.Error(error.CustomError{
				Message:    err.Error(),
				StatusCode: http.StatusBadRequest,
			})
			return
		}

		defer file.Close()

		// Загружаем аватарку
		avatar, err := srv.UploadAvatar(c.Request.Context(), userID, &file, fileHeader)

		// Если ошибка, возвращаем 500
		if err != nil {
			_ = c.Error(error.CustomError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}

		// Возвращаем созданный аватар
		c.JSON(http.StatusCreated, avatar)
	}
}
