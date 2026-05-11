package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/gin-gonic/gin"
)

// ErrorHandler captures error and returns a consistent JSON error response
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Process other handlers

		if len(c.Errors) > 0 {
			// Iterate through collected error
			for _, ginErr := range c.Errors {
				var customErr error.CustomError
				if errors.As(ginErr.Err, &customErr) {
					if strings.HasPrefix(c.Request.URL.Path, "/api/") {
						if customErr.StatusCode != http.StatusConflict {
							c.JSON(customErr.StatusCode, gin.H{"error": customErr.Message})
						} else {
							c.JSON(customErr.StatusCode, gin.H{"result": customErr.Message})
						}
					} else {
						c.String(customErr.StatusCode, customErr.Message)
					}
					return
				}
			}
		}
	}
}
