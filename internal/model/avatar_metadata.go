package model

import "time"

// AvatarThumbnailPublic элемент списка thumbnails в метаданных API.
type AvatarThumbnailPublic struct {
	Size string `json:"size"`
	URL  string `json:"url"`
}

// AvatarMetadata модель метаданных аватарки (упрощенная модель)
type AvatarMetadata struct {
	ID         string                  `json:"id"`
	UserID     string                  `json:"user_id"`
	FileName   string                  `json:"file_name"`
	MimeType   string                  `json:"mime_type"`
	Size       int64                   `json:"size"`
	Dimensions *ImageDimensions        `json:"dimensions,omitempty"`
	Thumbnails []AvatarThumbnailPublic `json:"thumbnails,omitempty"`
	CreatedAt  *time.Time              `json:"created_at,omitempty"`
	UpdatedAt  *time.Time              `json:"updated_at,omitempty"`
}
