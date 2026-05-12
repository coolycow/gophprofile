package model

// ImageDimensions размеры оригинала изображения (хранится в JSONB dimensions).
type ImageDimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}
