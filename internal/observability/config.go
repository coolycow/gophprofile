package observability

import "github.com/coolycow/gophprofile/internal/config"

// Config настройки инициализации наблюдаемости.
type Config struct {
	OtelEnabled     bool
	OtelEndpoint    string
	OtelServiceName string
	MetricsAddr     string
	LogFormat       string
	LogLevel        string
}

// FromServer собирает конфиг наблюдаемости из настроек HTTP-сервера.
func FromServer(cfg *config.ConfigServer) Config {
	return Config{
		OtelEnabled:     cfg.Observability.OtelEnabled,
		OtelEndpoint:    cfg.Observability.OtelEndpoint,
		OtelServiceName: cfg.Observability.OtelServiceName,
		MetricsAddr:     cfg.Observability.MetricsAddr,
		LogFormat:       cfg.Observability.LogFormat,
		LogLevel:        cfg.LogLevel,
	}
}

// FromWorker собирает конфиг наблюдаемости из настроек воркера.
func FromWorker(cfg *config.ConfigWorker) Config {
	return Config{
		OtelEnabled:     cfg.Observability.OtelEnabled,
		OtelEndpoint:    cfg.Observability.OtelEndpoint,
		OtelServiceName: cfg.Observability.OtelServiceName,
		MetricsAddr:     cfg.Observability.MetricsAddr,
		LogFormat:       cfg.Observability.LogFormat,
		LogLevel:        cfg.LogLevel,
	}
}
