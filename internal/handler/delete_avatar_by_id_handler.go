package handler

import (
	"net/http"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// DeleteAvatarByIDHandler удаляет аватарку по ID
func DeleteAvatarByIDHandler(srv service.AvatarService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем ID аватара из параметров запроса
		avatarID := c.Param("avatar_id")

		// Удаляем аватарку по ID
		err := srv.DeleteAvatarByID(c.Request.Context(), avatarID)

		// Если ошибка, возвращаем 500
		if err != nil {
			_ = c.Error(error.CustomError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}

		// Возвращаем успешный ответ
		c.JSON(http.StatusOK, gin.H{"message": "Avatar deleted successfully"})
	}
}
