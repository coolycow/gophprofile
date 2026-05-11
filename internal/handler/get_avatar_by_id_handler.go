package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// GetAvatarByIDHandler получает аватарку по ID
func GetAvatarByIDHandler(srv service.AvatarService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем ID аватара из параметров запроса
		avatarID := c.Param("avatar_id")

		// Получаем аватарку по ID
		avatar, err := srv.GetAvatarByID(c.Request.Context(), avatarID)

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
