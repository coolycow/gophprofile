package observability

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const tracerName = "gophprofile"

var shutdownFn func(context.Context) error

// Tracer возвращает именованный tracer приложения.
func Tracer() string {
	return tracerName
}

// InitTracing настраивает OTLP-экспорт трейсов и глобальный propagator.
func InitTracing(ctx context.Context, cfg Config) error {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if !cfg.OtelEnabled {
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		return nil
	}

	endpoint := strings.TrimSpace(cfg.OtelEndpoint)
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return fmt.Errorf("otlp trace exporter: %w", err)
	}

	serviceName := strings.TrimSpace(cfg.OtelServiceName)
	if serviceName == "" {
		serviceName = "gophprofile"
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return fmt.Errorf("otel resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	shutdownFn = tp.Shutdown
	return nil
}

// Shutdown сбрасывает буфер трейсов перед завершением процесса.
func Shutdown(ctx context.Context) error {
	if shutdownFn == nil {
		return nil
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return shutdownFn(shutdownCtx)
}
