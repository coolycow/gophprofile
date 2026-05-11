package handler

import (
	"net/http"

	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// DeleteAvatarByIDHandler удаляет аватарку по ID (требуется X-User-ID = владелец).
func DeleteAvatarByIDHandler(srv service.AvatarService) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID := c.GetString("user_id")
		avatarID := c.Param("avatar_id")

		err := srv.DeleteAvatarByID(c.Request.Context(), callerID, avatarID)
		if err != nil {
			pushServiceError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}
