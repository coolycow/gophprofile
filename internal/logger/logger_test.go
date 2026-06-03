package logger

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Неверный уровень логирования
func TestInitialize_invalidLevel(t *testing.T) {
	saved := Log
	t.Cleanup(func() { Log = saved })

	Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	before := Log

	err := Initialize("not-a-valid-level", "console")
	require.NoError(t, err)
	assert.NotSame(t, before, Log)
}

// Верный уровень логирования
func TestInitialize_success(t *testing.T) {
	saved := Log
	t.Cleanup(func() { Log = saved })

	Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	before := Log

	err := Initialize("info", "json")
	require.NoError(t, err)
	assert.NotSame(t, before, Log)
}

// Верные уровни логирования
func TestInitialize_validLevels(t *testing.T) {
	valid := []string{
		"debug",
		"info",
		"warn",
		"error",
		"INFO",
		"DEBUG",
		"WARN",
		"ERROR",
	}

	for _, level := range valid {
		t.Run(level, func(t *testing.T) {
			saved := Log
			t.Cleanup(func() { Log = saved })

			Log = slog.New(slog.NewTextHandler(io.Discard, nil))
			before := Log

			err := Initialize(level, "console")
			require.NoError(t, err)
			assert.NotSame(t, before, Log)
		})
	}
}
