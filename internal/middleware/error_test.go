package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	profileError "github.com/coolycow/gophprofile/internal/error"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ErrorHandler отдаёт JSON с телом error для /api/ и строку иначе.
func TestErrorHandler_CustomError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("json api path bad request", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := gin.New()
		r.Use(ErrorHandler())
		r.GET("/api/x", func(c *gin.Context) {
			_ = c.Error(profileError.CustomError{Message: "bad", StatusCode: http.StatusBadRequest})
		})
		req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "bad")
	})

	t.Run("json api forbidden with details", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := gin.New()
		r.Use(ErrorHandler())
		r.GET("/api/forbidden", func(c *gin.Context) {
			_ = c.Error(profileError.CustomError{
				Message:    "Forbidden",
				Details:    "You can only delete your own avatars",
				StatusCode: http.StatusForbidden,
			})
		})
		req := httptest.NewRequest(http.MethodGet, "/api/forbidden", nil)
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "Forbidden")
		assert.Contains(t, w.Body.String(), "You can only delete your own avatars")
	})

	t.Run("non api string body", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := gin.New()
		r.Use(ErrorHandler())
		r.GET("/health", func(c *gin.Context) {
			_ = c.Error(profileError.CustomError{Message: "oops", StatusCode: http.StatusForbidden})
		})
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusForbidden, w.Code)
		assert.Equal(t, "oops", w.Body.String())
	})
}
