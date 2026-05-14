package rabbitmq

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// ConsumerTag по умолчанию для подписчиков очереди (можно переопределить при нескольких инстансах).
const ConsumerTagDefault = "gophprofile-worker"

// ConsumeAvatarJobs регистрирует consumer для очереди обработки аватаров с prefetch prefetchCount.
func ConsumeAvatarJobs(ch *amqp.Channel, prefetchCount int, consumerTag string) (<-chan amqp.Delivery, error) {
	if err := ch.Qos(prefetchCount, 0, false); err != nil {
		return nil, err
	}

	if consumerTag == "" {
		consumerTag = ConsumerTagDefault
	}

	return ch.Consume(
		QueueAvatarJobs,
		consumerTag,
		false,
		false,
		false,
		false,
		nil,
	)
}
