package model

import "time"

// Avatar модель аватара
type Avatar struct {
	ID               string           `json:"id"`
	UserID           string           `json:"user_id"`
	FileName         string           `json:"file_name"`
	MimeType         string           `json:"mime_type"`
	SizeBytes        int64            `json:"size_bytes"`
	S3Key            string           `json:"s3_key"`
	ThumbnailS3Keys  StringSliceJSON  `json:"thumbnail_s3_keys,omitempty"`
	Dimensions       *ImageDimensions `json:"dimensions,omitempty"`
	UploadStatus     string           `json:"upload_status"`
	ProcessingStatus string           `json:"processing_status"`
	CreatedAt        *time.Time       `json:"created_at,omitempty"`
	UpdatedAt        *time.Time       `json:"updated_at,omitempty"`
	DeletedAt        *time.Time       `json:"deleted_at,omitempty"`
}

// BuildAvatarMetadata собирает публичные метаданные (URL миниатюр через resolve).
func (a *Avatar) BuildAvatarMetadata(resolveThumbnailURL func(s3Key string) string) *AvatarMetadata {
	// Если аватар пустой, то возвращаем nil
	if a == nil {
		return nil
	}

	// Если функция для генерации URL миниатюр не задана, то используем пустую функцию
	resolve := resolveThumbnailURL
	if resolve == nil {
		resolve = func(string) string { return "" }
	}

	// Генерируем список миниатюр
	thumbs := make([]AvatarThumbnailPublic, 0, len(a.ThumbnailS3Keys))
	for _, key := range []string(a.ThumbnailS3Keys) {
		if key == "" {
			continue
		}

		// Получаем размер миниатюры из ключа
		label := ThumbnailSizeLabelFromS3Key(key)
		if label == "" {
			continue
		}

		// Добавляем миниатюру в список
		thumbs = append(thumbs, AvatarThumbnailPublic{
			Size: label,
			URL:  resolve(key),
		})
	}

	// Возвращаем метаданные аватарки
	return &AvatarMetadata{
		ID:         a.ID,
		UserID:     a.UserID,
		FileName:   a.FileName,
		MimeType:   a.MimeType,
		Size:       a.SizeBytes,
		Dimensions: a.Dimensions,
		Thumbnails: thumbs,
		CreatedAt:  a.CreatedAt,
		UpdatedAt:  a.UpdatedAt,
	}
}
