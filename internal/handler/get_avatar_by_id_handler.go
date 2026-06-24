package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/service"
	"github.com/gin-gonic/gin"
)

// GetAvatarByIDHandler получает аватарку по ID
// @Summary Get avatar by ID
// @Tags avatars
// @Produce image/jpeg,image/png,image/webp
// @Param avatar_id path string true "Avatar UUID"
// @Param size query string false "Thumbnail size" Enums(100x100,300x300,original)
// @Param format query string false "Output format" Enums(jpeg,png,webp)
// @Success 200 {file} binary
// @Failure 404 {object} map[string]string
// @Router /api/v1/avatars/{avatar_id} [get]
func GetAvatarByIDHandler(srv service.AvatarService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем ID аватара из параметров запроса
		avatarID := c.Param("avatar_id")

		// Проверяем, является ли avatarID UUID
		if err := ValidateUUID(avatarID); err != nil {
			_ = c.Error(error.CustomError{
				Message:    "Invalid UUID",
				StatusCode: http.StatusBadRequest,
			})
			return
		}

		// Получаем аватарку по ID
		avatar, err := srv.GetAvatarByID(c.Request.Context(), avatarID)

		// Если ошибка, возвращаем 500
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				_ = c.Error(error.CustomError{
					Message:    "Avatar not found",
					StatusCode: http.StatusNotFound,
				})
				return
			}

			_ = c.Error(error.CustomError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}

		// Если аватарка не найдена, возвращаем 404
		if avatar == nil {
			_ = c.Error(error.CustomError{
				Message:    "Avatar not found",
				StatusCode: http.StatusNotFound,
			})
			return
		}

		// Отдаём аватарку из хранилища
		streamAvatarResponse(c, srv, avatar)
	}
}
