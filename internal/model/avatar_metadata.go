package model

import "time"

// AvatarMetadata модель метаданных аватарки
type AvatarMetadata struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	AvatarID  string         `json:"avatar_id"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt *time.Time     `json:"created_at,omitempty"`
	UpdatedAt *time.Time     `json:"updated_at,omitempty"`
	DeletedAt *time.Time     `json:"deleted_at,omitempty"`
}
