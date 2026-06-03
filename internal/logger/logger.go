// Package logger предоставляет глобальный логер приложения.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// Log будет доступен всему коду как синглтон.
// Никакой код навыка, кроме функции Initialize, не должен модифицировать эту переменную.
// По умолчанию установлен no-op-логер, который не выводит никаких сообщений.
var Log *slog.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

// Initialize инициализирует синглтон логера с необходимым уровнем логирования.
func Initialize(level, format string) error {
	lvl, err := parseLevel(level)
	if err != nil {
		return err
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler
	if strings.EqualFold(strings.TrimSpace(format), "json") {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	zl := slog.New(handler)
	// устанавливаем синглтон
	Log = zl
	slog.SetDefault(zl)

	return nil
}

func parseLevel(level string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, nil
	}
}

// FromContext возвращает логер с trace_id из context (если span активен).
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return Log
	}
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return Log
	}
	return Log.With(
		"trace_id", sc.TraceID().String(),
		"span_id", sc.SpanID().String(),
	)
}
