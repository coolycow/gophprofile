package handler

import (
	"net/http"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// GetAvatarsByUserIDHandler получает список аватарок пользователя
func GetAvatarsByUserIDHandler(srv service.AvatarService) gin.HandlerFunc {
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

		// Если аватарок не найдено, возвращаем 404
		if len(avatars) == 0 {
			_ = c.Error(error.CustomError{
				Message:    "No avatars found",
				StatusCode: http.StatusNotFound,
			})
			return
		}

		// Возвращаем список аватарок пользователя
		c.JSON(http.StatusOK, avatars)
	}
}
