package model

import "time"

// AvatarUploadResponse тело ответа POST /api/v1/avatars (как в ТЗ).
type AvatarUploadResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
