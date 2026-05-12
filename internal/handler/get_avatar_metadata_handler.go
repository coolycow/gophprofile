package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// GetAvatarMetadataHandler получает метаданные аватарки по ID
func GetAvatarMetadataHandler(srv service.AvatarService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем ID аватара из параметров запроса
		avatarID := c.Param("avatar_id")

		// Проверяем, является ли avatarID UUID
		if err := ValidateUUID(avatarID); err != nil {
			_ = c.Error(error.CustomError{
				Message:    "Invalid UUID",
				StatusCode: http.StatusBadRequest,
			})
			return
		}

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
				Message:    "Metadata not found",
				StatusCode: http.StatusNotFound,
			})
			return
		}

		// Возвращаем метаданные аватарки
		c.JSON(http.StatusOK, avatar.GetMetadata())
	}
}
