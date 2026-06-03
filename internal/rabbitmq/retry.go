package rabbitmq

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/coolycow/gophprofile/internal/observability"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/google/uuid"
)

const headerRetryCount = "x-retry-count"

// MaxAvatarJobRetries число повторных попыток после первой неудачи (экспоненциальная задержка + republish).
const MaxAvatarJobRetries = 5

// HeaderRetryCount читает счётчик повторов из заголовков доставки.
func HeaderRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	v, ok := headers[headerRetryCount]
	if !ok {
		return 0
	}
	switch t := v.(type) {
	case int:
		return t
	case int32:
		return int(t)
	case int64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(t)
		return n
	default:
		return 0
	}
}

// IsTransientAvatarJobError эвристика: временные сбои можно повторить, постоянные — нет.
func IsTransientAvatarJobError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "decode") || strings.Contains(s, "invalid") {
		return false
	}
	if strings.Contains(s, "nosuchkey") {
		return false
	}
	return true
}

// RepublishAvatarJob публикует то же тело в exchange с увеличенным x-retry-count.
func RepublishAvatarJob(ctx context.Context, pub *amqp.Channel, body []byte, job *AvatarJobMessage, nextRetry int) error {
	if job == nil {
		return fmt.Errorf("nil job")
	}
	rk := RoutingKeyForJob(job)
	if rk == "" {
		return fmt.Errorf("empty routing key for job type %q", job.Type)
	}
	headers := amqp.Table{headerRetryCount: nextRetry}
	headers = observability.InjectAMQPHeaders(ctx, headers)
	return pub.PublishWithContext(ctx, ExchangeAvatars, rk, false, false, amqp.Publishing{
		ContentType:   "application/json",
		DeliveryMode:  amqp.Persistent,
		MessageId:     uuid.NewString(),
		Headers:       headers,
		Body:          body,
	})
}

// RetryBackoffDuration задержка перед republish.
func RetryBackoffDuration(retry int) time.Duration {
	if retry < 0 {
		retry = 0
	}
	if retry > 8 {
		retry = 8
	}
	return time.Duration(1<<retry) * 100 * time.Millisecond
}
