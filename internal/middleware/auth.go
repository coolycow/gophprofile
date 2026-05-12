package middleware

import (
	"net/http"
	"strings"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HeaderXUserID имя заголовка с идентификатором пользователя (шлюз / BFF).
const HeaderXUserID = "X-User-ID"

// RequireXUserID требует непустой X-User-ID и сохраняет его в контексте как user_id.
func RequireXUserID(userSvc service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем X-User-ID из заголовка
		uid := strings.TrimSpace(c.GetHeader(HeaderXUserID))

		// Если X-User-ID пустой, возвращаем ошибку 401
		if uid == "" {
			_ = c.Error(error.CustomError{
				Message:    "Unauthorized",
				Details:    "X-User-ID header is required",
				StatusCode: http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Если X-User-ID не является UUID, возвращаем ошибку 401
		if _, err := uuid.Parse(uid); err != nil {
			_ = c.Error(error.CustomError{
				Message:    "Unauthorized",
				Details:    "X-User-ID is not a valid UUID",
				StatusCode: http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Проверяем существование пользователя в базе данных
		user, err := userSvc.GetUserByID(c.Request.Context(), uid)
		if err != nil {
			_ = c.Error(error.CustomError{
				Message:    "Unauthorized",
				Details:    "User not found",
				StatusCode: http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Если пользователь не найден, возвращаем ошибку 401
		if user == nil {
			_ = c.Error(error.CustomError{
				Message:    "Unauthorized",
				Details:    "User not found",
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
