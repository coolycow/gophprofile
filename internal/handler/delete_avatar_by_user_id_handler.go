package handler

import (
	"net/http"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// DeleteAvatarByUserIDHandler удаляет аватарку по ID пользователя
func DeleteAvatarByUserIDHandler(srv service.AvatarService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем ID аватара из параметров запроса
		userID := c.Param("user_id")

		// Удаляем аватарку по ID
		err := srv.DeleteAvatarByUserID(c.Request.Context(), userID)

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
