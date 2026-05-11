package model

import "time"

// Avatar модель аватара
type Avatar struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	FileName         string     `json:"file_name"`
	MimeType         string     `json:"mime_type"`
	SizeBytes        int64      `json:"size_bytes"`
	S3Key            string     `json:"s3_key"`
	ThumbnailS3Keys  []string   `json:"thumbnail_s3_keys,omitempty"`
	UploadStatus     string     `json:"upload_status"`
	ProcessingStatus string     `json:"processing_status"`
	CreatedAt        *time.Time `json:"created_at,omitempty"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
}

// GetMetadata получает метаданные аватарки
func (a *Avatar) GetMetadata() *AvatarMetadata {
	return &AvatarMetadata{
		ID:              a.ID,
		UserID:          a.UserID,
		FileName:        a.FileName,
		MimeType:        a.MimeType,
		SizeBytes:       a.SizeBytes,
		ThumbnailS3Keys: a.ThumbnailS3Keys,
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
	}
}
