package handler

import (
	"net/http"

	apperror "github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/model"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)
// RegisterHandler регистрирует пользователя и возвращает токены.
func RegisterHandler(srv service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.UserRegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			_ = c.Error(apperror.CustomError{
				Message:    err.Error(),
				StatusCode: http.StatusBadRequest,
			})
			return
		}

		resp, err := srv.RegisterWithAuth(c.Request.Context(), req)
		if err != nil {
			pushServiceError(c, err)
			return
		}

		c.JSON(http.StatusCreated, resp)
	}
}

// LoginHandler выполняет вход и возвращает токены.
func LoginHandler(srv service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			_ = c.Error(apperror.CustomError{
				Message:    err.Error(),
				StatusCode: http.StatusBadRequest,
			})
			return
		}

		resp, err := srv.LoginWithAuth(c.Request.Context(), req)
		if err != nil {
			pushServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

// RefreshHandler обновляет пару токенов по refresh_token.
func RefreshHandler(srv service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.RefreshTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			_ = c.Error(apperror.CustomError{
				Message:    err.Error(),
				StatusCode: http.StatusBadRequest,
			})
			return
		}

		resp, err := srv.RefreshWithAuth(c.Request.Context(), req.RefreshToken)
		if err != nil {
			pushServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}
