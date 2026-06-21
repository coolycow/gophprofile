package config

import (
	"os"
	"strconv"
	"strings"
)

// ObservabilityConfig общие настройки наблюдаемости для server и worker.
type ObservabilityConfig struct {
	OtelEnabled     bool   `env:"OTEL_ENABLED" json:"otel_enabled,omitempty"`                       // Включить экспорт трейсов в OTLP
	OtelEndpoint    string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" json:"otel_exporter_otlp_endpoint,omitempty"` // Адрес OTLP collector (Jaeger)
	OtelServiceName string `env:"OTEL_SERVICE_NAME" json:"otel_service_name,omitempty"`           // Имя сервиса в трейсах
	MetricsAddr     string `env:"METRICS_ADDR" json:"metrics_addr,omitempty"`                       // Адрес HTTP-сервера /metrics
	LogFormat       string `env:"LOG_FORMAT" json:"log_format,omitempty"`                           // json или console
}

type fileObservabilityConfig struct {
	OtelEnabled     *bool   `json:"otel_enabled"`
	OtelEndpoint    *string `json:"otel_exporter_otlp_endpoint"`
	OtelServiceName *string `json:"otel_service_name"`
	MetricsAddr     *string `json:"metrics_addr"`
	LogFormat       *string `json:"log_format"`
}

func defaultObservabilityConfig() ObservabilityConfig {
	return ObservabilityConfig{
		OtelEnabled:  false,
		OtelEndpoint: "localhost:4317",
		MetricsAddr:  ":9090",
		LogFormat:    "console",
	}
}

func applyObservabilityEnv(cfg *ObservabilityConfig) {
	if v, ok := os.LookupEnv("OTEL_ENABLED"); ok {
		cfg.OtelEnabled, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT"); ok {
		cfg.OtelEndpoint = strings.TrimSpace(v)
	}
	if v, ok := os.LookupEnv("OTEL_SERVICE_NAME"); ok {
		cfg.OtelServiceName = strings.TrimSpace(v)
	}
	if v, ok := os.LookupEnv("METRICS_ADDR"); ok {
		cfg.MetricsAddr = strings.TrimSpace(v)
	}
	if v, ok := os.LookupEnv("LOG_FORMAT"); ok {
		cfg.LogFormat = strings.TrimSpace(v)
	}
}

func mergeObservabilityFromFile(dst *ObservabilityConfig, fc *fileObservabilityConfig) {
	if fc == nil {
		return
	}
	if fc.OtelEnabled != nil {
		dst.OtelEnabled = *fc.OtelEnabled
	}
	if fc.OtelEndpoint != nil {
		dst.OtelEndpoint = *fc.OtelEndpoint
	}
	if fc.OtelServiceName != nil {
		dst.OtelServiceName = *fc.OtelServiceName
	}
	if fc.MetricsAddr != nil {
		dst.MetricsAddr = *fc.MetricsAddr
	}
	if fc.LogFormat != nil {
		dst.LogFormat = *fc.LogFormat
	}
}
