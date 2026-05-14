package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Неверный уровень логирования
func TestInitialize_invalidLevel(t *testing.T) {
	saved := Log
	t.Cleanup(func() { Log = saved })

	Log = zap.NewNop()
	before := Log

	err := Initialize("not-a-valid-level")
	require.Error(t, err)
	assert.Same(t, before, Log)
}

// Верный уровень логирования
func TestInitialize_success(t *testing.T) {
	saved := Log
	t.Cleanup(func() { Log = saved })

	Log = zap.NewNop()
	before := Log

	err := Initialize("info")
	require.NoError(t, err)
	assert.NotSame(t, before, Log)
	require.NoError(t, Log.Sync())
}

// Верные уровни логирования
func TestInitialize_validLevels(t *testing.T) {
	valid := []string{
		"debug",
		"info",
		"warn",
		"error",
		"dpanic",
		"INFO",
		"DEBUG",
		"INFO",
		"WARN",
		"ERROR",
		"DPANIC",
		"PANIC",
	}

	for _, level := range valid {
		t.Run(level, func(t *testing.T) {
			saved := Log
			t.Cleanup(func() { Log = saved })

			Log = zap.NewNop()
			before := Log

			err := Initialize(level)
			require.NoError(t, err)
			assert.NotSame(t, before, Log)
			require.NoError(t, Log.Sync())
		})
	}
}
