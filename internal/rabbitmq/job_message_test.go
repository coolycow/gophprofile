package rabbitmq

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeAvatarJob_process(t *testing.T) {
	raw, err := json.Marshal(AvatarJobMessage{Type: AvatarJobTypeProcess, AvatarID: "a1"})
	require.NoError(t, err)

	msg, err := DecodeAvatarJob(raw)
	require.NoError(t, err)
	require.Equal(t, AvatarJobTypeProcess, msg.Type)
	require.Equal(t, "a1", msg.AvatarID)
}

func TestDecodeAvatarJob_legacyShapeRejected(t *testing.T) {
	// Старый формат без поля type — должен отклоняться.
	_, err := DecodeAvatarJob([]byte(`{"avatar_id":"x"}`))
	require.Error(t, err)
}

func TestDecodeAvatarJob_deleteByS3Key_withThumbnails(t *testing.T) {
	raw, err := json.Marshal(AvatarJobMessage{
		Type:            AvatarJobTypeDeleteByS3Key,
		S3Key:           "user/original.jpg",
		ThumbnailS3Keys: []string{"user/thumbnails/id_100.jpg", "user/thumbnails/id_300.jpg"},
	})
	require.NoError(t, err)

	msg, err := DecodeAvatarJob(raw)
	require.NoError(t, err)
	require.Equal(t, AvatarJobTypeDeleteByS3Key, msg.Type)
	require.Equal(t, "user/original.jpg", msg.S3Key)
	require.Equal(t, []string{"user/thumbnails/id_100.jpg", "user/thumbnails/id_300.jpg"}, msg.ThumbnailS3Keys)
}
