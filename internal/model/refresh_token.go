package model

import "time"

// RefreshToken строка в таблице refresh_tokens (хранится только хэш непрозрачного токена).
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash []byte
	ExpiresAt time.Time // не используем указатель, т.к. в БД не может быть NULL
	CreatedAt time.Time // не используем указатель, т.к. в БД не может быть NULL
}
