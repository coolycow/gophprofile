package handler

import (
	"net/http"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// GetUserAvatarsHandler получает список аватарок пользователя
func GetUserAvatarsHandler(srv service.AvatarService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем ID пользователя из параметров запроса
		userID := c.Param("user_id")

		// Получаем список аватарок пользователя
		avatars, err := srv.GetUserAvatars(c.Request.Context(), userID)

		// Если ошибка, возвращаем 500
		if err != nil {
			_ = c.Error(error.CustomError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}

		// Возвращаем список аватарок пользователя
		c.JSON(http.StatusOK, avatars)
	}
}
