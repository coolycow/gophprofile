package observability

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// amqpHeaderCarrier адаптирует amqp.Table для W3C propagation.
type amqpHeaderCarrier amqp.Table

func (c amqpHeaderCarrier) Get(key string) string {
	if c == nil {
		return ""
	}
	v, ok := c[key]
	if !ok {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	default:
		return ""
	}
}

func (c amqpHeaderCarrier) Set(key, value string) {
	amqp.Table(c)[key] = value
}

func (c amqpHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// InjectAMQPHeaders записывает trace context в заголовки AMQP-сообщения.
func InjectAMQPHeaders(ctx context.Context, headers amqp.Table) amqp.Table {
	if headers == nil {
		headers = amqp.Table{}
	}
	otel.GetTextMapPropagator().Inject(ctx, amqpHeaderCarrier(headers))
	return headers
}

// ExtractAMQPContext восстанавливает parent span из заголовков AMQP-сообщения.
func ExtractAMQPContext(ctx context.Context, headers amqp.Table) context.Context {
	if len(headers) == 0 {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, amqpHeaderCarrier(headers))
}

// MapCarrier для произвольных string map (тесты и вспомогательные сценарии).
type MapCarrier map[string]string

func (c MapCarrier) Get(key string) string { return c[key] }
func (c MapCarrier) Set(key, value string) { c[key] = value }
func (c MapCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// InjectMap записывает trace context в map[string]string.
func InjectMap(ctx context.Context, carrier MapCarrier) {
	if carrier == nil {
		return
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(carrier))
}
