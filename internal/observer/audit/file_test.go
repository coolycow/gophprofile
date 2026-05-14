package audit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coolycow/gophprofile/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FileReceiver пишет JSON-строки и корректно закрывается.
func TestFileReceiver_SendAndClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	r, err := NewFileReceiver(path)
	require.NoError(t, err)

	ev := &model.Audit{Action: "login", UserID: "u1"}
	require.NoError(t, r.Send(ev))
	require.NoError(t, r.Close())

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(b), "login")

	require.NoError(t, r.Close())
}
