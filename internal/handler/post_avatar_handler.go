package handler

import (
	"net/http"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/observer/audit"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// PostAvatarHandler загружает аватарку
func PostAvatarHandler(srv service.AvatarService, auditNotifier *audit.Notifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		file, fileHeader, err := c.Request.FormFile("file")

		if err != nil {
			_ = c.Error(error.CustomError{
				Message:    err.Error(),
				StatusCode: http.StatusBadRequest,
			})
			return
		}

		defer file.Close()

		avatar, err := srv.UploadAvatar(c.Request.Context(), userID, &file, fileHeader)

		if err != nil {
			_ = c.Error(error.CustomError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}

		c.JSON(http.StatusCreated, avatar)
	}
}
