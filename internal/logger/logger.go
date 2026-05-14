// Package logger предоставляет глобальный логер приложения.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log будет доступен всему коду как синглтон.
// Никакой код навыка, кроме функции Initialize, не должен модифицировать эту переменную.
// По умолчанию установлен no-op-логер, который не выводит никаких сообщений.
var Log *zap.Logger = zap.NewNop()

// Initialize инициализирует синглтон логера с необходимым уровнем логирования.
func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)

	if err != nil {
		return err
	}

	// создаём новую конфигурацию логера
	cfg := zap.NewProductionConfig()

	// человекочитаемый вывод в консоль с цветным уровнем (ANSI)
	cfg.Encoding = "console"
	cfg.EncoderConfig.EncodeLevel = zapcore.LowercaseColorLevelEncoder

	// устанавливаем уровень
	cfg.Level = lvl
	// создаём логер на основе конфигурации

	zl, err := cfg.Build()

	if err != nil {
		return err
	}

	// устанавливаем синглтон
	Log = zl

	return nil
}
