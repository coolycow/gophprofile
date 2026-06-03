package observability

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type ctxKey struct{}

// WithLogger сохраняет логер с полем service в context.
func WithLogger(ctx context.Context, serviceName string) context.Context {
	l := slog.Default().With("service", serviceName)
	return context.WithValue(ctx, ctxKey{}, l)
}

// LoggerFromContext возвращает логер с trace_id и span_id из активного span.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	base := slog.Default()
	if v := ctx.Value(ctxKey{}); v != nil {
		if l, ok := v.(*slog.Logger); ok && l != nil {
			base = l
		}
	}

	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return base
	}
	return base.With(
		"trace_id", sc.TraceID().String(),
		"span_id", sc.SpanID().String(),
	)
}
