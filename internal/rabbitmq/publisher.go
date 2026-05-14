package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// AvatarJobPublisher отправляет задания в exchange avatars.exchange (topic).
type AvatarJobPublisher struct {
	mu sync.Mutex
	ch *amqp.Channel
}

// NewAvatarJobPublisher инициализирует Publisher для работы с аватарками.
func NewAvatarJobPublisher(ch *amqp.Channel) *AvatarJobPublisher {
	return &AvatarJobPublisher{ch: ch}
}

// publishJob публикует задание в exchange с routing key и уникальным MessageId.
func (p *AvatarJobPublisher) publishJob(ctx context.Context, msg AvatarJobMessage) error {
	// Проверяем, согласованы ли тип задания и поля
	if err := msg.Validate(); err != nil {
		return err
	}

	// Сериализуем тело сообщения
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Получаем routing key для задания
	rk := RoutingKeyForJob(&msg)
	if rk == "" {
		return fmt.Errorf("rabbitmq: empty routing key for job type %q", msg.Type)
	}

	// Публикуем задание в exchange с routing key и уникальным MessageId.
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ch.PublishWithContext(ctx, ExchangeAvatars, rk, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    uuid.NewString(),
		Body:         body,
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
