package middleware

import (
	"net/http"
	"strings"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// HeaderXUserID имя заголовка с идентификатором пользователя (шлюз / BFF).
const HeaderXUserID = "X-User-ID"

// RequireXUserID требует непустой X-User-ID и сохраняет его в контексте как user_id.
func RequireXUserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := strings.TrimSpace(c.GetHeader(HeaderXUserID))
		if uid == "" {
			_ = c.Error(error.CustomError{
				Message:    "Unauthorized",
				Details:    "X-User-ID header is required",
				StatusCode: http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		c.Set("user_id", uid)
		c.Next()
	}
}

// RequireAuth проверяет JWT в Authorization (Bearer) и записывает user_id в контекст Gin.
func RequireAuth(userSvc service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		userID, err := userSvc.GetUserIDFromAuthToken(token)
		if err != nil {
			_ = c.Error(error.CustomError{
				Message:    "unauthorized",
				StatusCode: http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}