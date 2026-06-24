package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/coolycow/gophprofile/internal/observability"
	"github.com/coolycow/gophprofile/internal/resilience"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// AvatarJobPublisher отправляет задания в exchange avatars.exchange (topic).
type AvatarJobPublisher struct {
	mu      sync.Mutex
	ch      *amqp.Channel
	breaker *gobreaker.CircuitBreaker
}

// NewAvatarJobPublisher инициализирует Publisher для работы с аватарками.
// breaker передаётся снаружи; nil — publish без circuit breaker.
func NewAvatarJobPublisher(ch *amqp.Channel, breaker *gobreaker.CircuitBreaker) *AvatarJobPublisher {
	return &AvatarJobPublisher{ch: ch, breaker: breaker}
}

// publishJob публикует задание в exchange с routing key и уникальным MessageId.
func (p *AvatarJobPublisher) publishJob(ctx context.Context, msg AvatarJobMessage) error {
	ctx, span := otel.Tracer(observability.Tracer()).Start(ctx, "rabbitmq.publish")
	defer span.End()

	// Проверяем, согласованы ли тип задания и поля
	if err := msg.Validate(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// Сериализуем тело сообщения
	body, err := json.Marshal(msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// Получаем routing key для задания
	rk := RoutingKeyForJob(&msg)
	if rk == "" {
		err = fmt.Errorf("rabbitmq: empty routing key for job type %q", msg.Type)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetAttributes(
		attribute.String("messaging.system", "rabbitmq"),
		attribute.String("messaging.destination", ExchangeAvatars),
		attribute.String("messaging.rabbitmq.routing_key", rk),
		attribute.String("job.type", string(msg.Type)),
	)

	headers := observability.InjectAMQPHeaders(ctx, amqp.Table{})

	// Публикуем задание в exchange с routing key и уникальным MessageId.
	p.mu.Lock()
	defer p.mu.Unlock()
	return resilience.ExecuteVoid(p.breaker, func() error {
		err = p.ch.PublishWithContext(ctx, ExchangeAvatars, rk, false, false, amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    uuid.NewString(),
			Headers:      headers,
			Body:         body,
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	})
}

// PublishAvatarProcessingJob ставит задание на обработку изображения.
func (p *AvatarJobPublisher) PublishAvatarProcessingJob(ctx context.Context, avatarID string) error {
	return p.publishJob(ctx, AvatarJobMessage{
		Type:     AvatarJobTypeProcess,
		AvatarID: avatarID,
	})
}

// PublishAvatarDeletionByS3KeyJob ставит задание на очистку S3: оригинал и миниатюры.
func (p *AvatarJobPublisher) PublishAvatarDeletionByS3KeyJob(ctx context.Context, s3Key string, thumbnailS3Keys []string) error {
	return p.publishJob(ctx, AvatarJobMessage{
		Type:            AvatarJobTypeDeleteByS3Key,
		S3Key:           s3Key,
		ThumbnailS3Keys: thumbnailS3Keys,
	})
}
