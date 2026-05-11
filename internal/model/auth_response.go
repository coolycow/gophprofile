package model

import "time"

// AuthResponse ответ авторизации
type AuthResponse struct {
	Salt         string     `json:"salt"`
	Token        string     `json:"token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresAt    *time.Time `json:"expires_at"`
}
