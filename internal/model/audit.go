package model

// Audit — событие аудита (без чувствительных полезных данных: только идентификаторы и тип действия).
type Audit struct {
	TS               int    `json:"ts"`                  // unix timestamp события
	Action           string `json:"action"`              // тип действия (см. пакет audit)
	UserID           string `json:"user_id,omitempty"`   // пользователь, от имени которого выполнено действие
	AvatarID         string `json:"avatar_id,omitempty"` // затронутый аватар
	ProcessingStatus string `json:"processing_status"`   // статус обработки аватара
}
