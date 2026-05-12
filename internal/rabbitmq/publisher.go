package rabbitmq

import (
	"context"
	"encoding/json"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// AvatarJobPublisher отправляет задания в очередь обработки аватаров.
type AvatarJobPublisher struct {
	mu sync.Mutex
	ch *amqp.Channel
}

// NewAvatarJobPublisher инициализирует Publisher для работы с аватарками.
func NewAvatarJobPublisher(ch *amqp.Channel) *AvatarJobPublisher {
	return &AvatarJobPublisher{ch: ch}
}

// publishJob публикует задание в очередь
func (p *AvatarJobPublisher) publishJob(ctx context.Context, msg AvatarJobMessage) error {
	// Проверка на валидность сообщения
	if err := msg.Validate(); err != nil {
		return err
	}

	// Сериализация тела сообщения
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Публикация сообщения в очередь
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ch.PublishWithContext(ctx, "", QueueAvatarJobs, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
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

// PublishAvatarDeletionByS3KeyJob ставит задание на очистку S3 по ключу.
func (p *AvatarJobPublisher) PublishAvatarDeletionByS3KeyJob(ctx context.Context, s3Key string) error {
	return p.publishJob(ctx, AvatarJobMessage{
		Type:  AvatarJobTypeDeleteByS3Key,
		S3Key: s3Key,
	})
}
