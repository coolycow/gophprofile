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

func TestDecodeAvatarJob_unknownType(t *testing.T) {
	_, err := DecodeAvatarJob([]byte(`{"type":"oops","avatar_id":"x"}`))
	require.Error(t, err)
}
