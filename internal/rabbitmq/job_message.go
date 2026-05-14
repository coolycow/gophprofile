package rabbitmq

import (
	"encoding/json"
	"fmt"
)

// AvatarJobType вид задания в очереди (дискriminator в JSON).
type AvatarJobType string

const (
	// AvatarJobTypeProcess постобработка загруженного аватара.
	AvatarJobTypeProcess AvatarJobType = "process"
	// AvatarJobTypeDeleteByS3Key удаление по s3_key (напр. ключи MinIO после DeleteAvatarByS3Key в репозитории).
	AvatarJobTypeDeleteByS3Key AvatarJobType = "delete_by_s3_key"
)

// AvatarJobMessage единый формат сообщения в очереди gophprofile.avatars.
type AvatarJobMessage struct {
	Type             AvatarJobType `json:"type"`
	AvatarID         string        `json:"avatar_id,omitempty"`
	UserID           string        `json:"user_id,omitempty"`
	S3Key            string        `json:"s3_key,omitempty"`
	ThumbnailS3Keys  []string      `json:"thumbnail_s3_keys,omitempty"`
}

// Validate проверяет согласованность type и полей.
func (m *AvatarJobMessage) Validate() error {
	if m.Type == "" {
		return fmt.Errorf("rabbitmq: missing job type")
	}
	switch m.Type {
	case AvatarJobTypeProcess:
		if m.AvatarID == "" {
			return fmt.Errorf("rabbitmq: job %s requires avatar_id", m.Type)
		}
	case AvatarJobTypeDeleteByS3Key:
		if m.S3Key == "" {
			return fmt.Errorf("rabbitmq: job %s requires s3_key", m.Type)
		}
	default:
		return fmt.Errorf("rabbitmq: unknown job type %q", m.Type)
	}
	return nil
}

// DecodeAvatarJob разбирает тело сообщения из очереди.
func DecodeAvatarJob(body []byte) (*AvatarJobMessage, error) {
	var msg AvatarJobMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, err
	}
	if err := msg.Validate(); err != nil {
		return nil, err
	}
	return &msg, nil
}
