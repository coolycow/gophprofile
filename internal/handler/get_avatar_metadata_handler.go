package handler

import (
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

		// Получаем метаданные аватарки по ID
		metadata, err := srv.GetAvatarMetadataByID(c.Request.Context(), avatarID)

		// Если ошибка, возвращаем 500
		if err != nil {
			_ = c.Error(error.CustomError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}

		// Если метаданные не найдены, возвращаем 404
		if metadata == nil {
			_ = c.Error(error.CustomError{
				Message:    "Metadata not found",
				StatusCode: http.StatusNotFound,
			})
			return
		}

		// Возвращаем метаданные аватарки
		c.JSON(http.StatusOK, metadata)
	}
}
