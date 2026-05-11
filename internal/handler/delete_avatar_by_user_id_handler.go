package handler

import (
	"net/http"

	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// DeleteAvatarByUserIDHandler удаляет аватар по user_id в пути; X-User-ID должен совпадать.
func DeleteAvatarByUserIDHandler(srv service.AvatarService) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID := c.GetString("user_id")
		pathUserID := c.Param("user_id")

		err := srv.DeleteAvatarByUserID(c.Request.Context(), callerID, pathUserID)
		if err != nil {
			pushServiceError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}
