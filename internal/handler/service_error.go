package handler

import (
	"errors"
	"net/http"

	apperror "github.com/coolycow/gophprofile/internal/error"
	"github.com/gin-gonic/gin"
)

// pushServiceError преобразует ошибку сервиса в ошибку Gin
func pushServiceError(c *gin.Context, svcErr error) {
	var ce apperror.CustomError
	if errors.As(svcErr, &ce) {
		_ = c.Error(ce)
		return
	}
	_ = c.Error(apperror.CustomError{
		Message:    svcErr.Error(),
		StatusCode: http.StatusInternalServerError,
	})
}
