package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// GetAvatarByUserIDHandler получает аватарку по ID пользователя
func GetAvatarByUserIDHandler(srv service.AvatarService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем ID пользователя из параметров запроса
		userID := c.Param("user_id")

		// Проверяем, является ли userID UUID
		if err := ValidateUUID(userID); err != nil {
			_ = c.Error(error.CustomError{
				Message:    "Invalid UUID",
				StatusCode: http.StatusBadRequest,
			})
			return
		}

		// Получаем аватарку по ID пользователя
		avatar, err := srv.GetAvatarByUserID(c.Request.Context(), userID)

		// Если ошибка, возвращаем 500
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				_ = c.Error(error.CustomError{
					Message:    "Avatar not found",
					StatusCode: http.StatusNotFound,
				})
				return
			}

			_ = c.Error(error.CustomError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}

		// Если аватарка не найдена, возвращаем 404
		if avatar == nil {
			_ = c.Error(error.CustomError{
				Message:    "Avatar not found",
				StatusCode: http.StatusNotFound,
			})
			return
		}

		// Возвращаем аватарку в виде файла
		c.File(avatar.S3Key)
	}
}
