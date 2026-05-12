package handler

import (
	"net/http"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// DeleteAvatarByUserIDHandler удаляет аватар по user_id в пути; X-User-ID должен совпадать.
func DeleteAvatarByUserIDHandler(srv service.AvatarService) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID := c.GetString("user_id")
		pathUserID := c.Param("user_id")

		// Проверяем, является ли pathUserID UUID
		if err := ValidateUUID(pathUserID); err != nil {
			_ = c.Error(error.CustomError{
				Message:    "Invalid UUID",
				StatusCode: http.StatusBadRequest,
			})
			return
		}

		// Удаляем аватарку по user_id в пути
		err := srv.DeleteAvatarByUserID(c.Request.Context(), callerID, pathUserID)
		if err != nil {
			pushServiceError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}
