package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// AvatarProcessingJob тело сообщения в очереди обработки аватаров.
type AvatarProcessingJob struct {
	AvatarID string `json:"avatar_id"`
}

// AvatarDeletionByIDJob тело сообщения в очереди удаления аватарки по ID.
type AvatarDeletionByIDJob struct {
	AvatarID string `json:"avatar_id"`
}

// AvatarDeletionByUserIDJob тело сообщения в очереди удаления аватарки по ID пользователя.
type AvatarDeletionByUserIDJob struct {
	UserID string `json:"user_id"`
}

// AvatarJobPublisher отправляет сообщения о необходимости обработать аватар.
type AvatarJobPublisher struct {
	mu sync.Mutex
	ch *amqp.Channel
}

// NewAvatarJobPublisher инициализирует Publisher для работы с аватарками
func NewAvatarJobPublisher(ch *amqp.Channel) *AvatarJobPublisher {
	return &AvatarJobPublisher{ch: ch}
}

// PublishAvatarProcessingJob публикует задание. Безопасен при параллельных HTTP-запросах (mutex на канал).
func (p *AvatarJobPublisher) PublishAvatarProcessingJob(ctx context.Context, avatarID string) error {
	// Проверка на пустоту avatar_id
	if avatarID == "" {
		return fmt.Errorf("rabbitmq: empty avatar_id")
	}

	// Сериализация тела сообщения
	body, err := json.Marshal(AvatarProcessingJob{AvatarID: avatarID})
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

// PublishAvatarDeletionByIDJob публикует задание на удаление аватарки
func (p *AvatarJobPublisher) PublishAvatarDeletionByIDJob(ctx context.Context, avatarID string) error {
	// Проверка на пустоту avatar_id
	if avatarID == "" {
		return fmt.Errorf("rabbitmq: empty avatar_id")
	}

	// Сериализация тела сообщения
	body, err := json.Marshal(AvatarDeletionByIDJob{AvatarID: avatarID})
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

// PublishAvatarDeletionByUserIDJob публикует задание на удаление аватарки
func (p *AvatarJobPublisher) PublishAvatarDeletionByUserIDJob(ctx context.Context, userID string) error {
	// Проверка на пустоту user_id
	if userID == "" {
		return fmt.Errorf("rabbitmq: empty user_id")
	}

	// Сериализация тела сообщения
	body, err := json.Marshal(AvatarDeletionByUserIDJob{UserID: userID})
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
