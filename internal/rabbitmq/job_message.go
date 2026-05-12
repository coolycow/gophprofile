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
	// AvatarJobTypeDeleteByAvatarID удаление объекта аватара в S3/БД после удаления строки по avatar_id на API.
	AvatarJobTypeDeleteByAvatarID AvatarJobType = "delete_by_avatar_id"
	// AvatarJobTypeDeleteByUserID удаление по user_id (напр. ключи MinIO после DeleteAvatarByUserID в репозитории).
	AvatarJobTypeDeleteByUserID AvatarJobType = "delete_by_user_id"
	// AvatarJobTypeDeleteByS3Key удаление по s3_key (напр. ключи MinIO после DeleteAvatarByS3Key в репозитории).
	AvatarJobTypeDeleteByS3Key AvatarJobType = "delete_by_s3_key"
)

// AvatarJobMessage единый формат сообщения в очереди gophprofile.avatars.
type AvatarJobMessage struct {
	Type     AvatarJobType `json:"type"`
	AvatarID string        `json:"avatar_id,omitempty"`
	UserID   string        `json:"user_id,omitempty"`
	S3Key    string        `json:"s3_key,omitempty"`
}

// Validate проверяет согласованность type и полей.
func (m *AvatarJobMessage) Validate() error {
	if m.Type == "" {
		return fmt.Errorf("rabbitmq: missing job type")
	}
	switch m.Type {
	case AvatarJobTypeProcess, AvatarJobTypeDeleteByAvatarID:
		if m.AvatarID == "" {
			return fmt.Errorf("rabbitmq: job %s requires avatar_id", m.Type)
		}
	case AvatarJobTypeDeleteByUserID:
		if m.UserID == "" {
			return fmt.Errorf("rabbitmq: job %s requires user_id", m.Type)
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
