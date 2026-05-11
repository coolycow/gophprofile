package model

import "time"

// AvatarMetadata модель метаданных аватарки (упрощенная модель)
type AvatarMetadata struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	FileName        string     `json:"file_name"`
	MimeType        string     `json:"mime_type"`
	SizeBytes       int64      `json:"size_bytes"`
	ThumbnailS3Keys []string   `json:"thumbnail_s3_keys,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}
