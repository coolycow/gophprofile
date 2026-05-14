package rabbitmq

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// ExchangeAvatars — topic exchange для заданий аватаров (как в ТЗ: direct/topic).
	ExchangeAvatars = "avatars.exchange"
	// RoutingKeyAvatarProcess — постобработка (миниатюры).
	RoutingKeyAvatarProcess = "avatar.process"
	// RoutingKeyAvatarDelete — удаление объектов в S3.
	RoutingKeyAvatarDelete = "avatar.delete"
	// QueueAvatarJobs — durable-очередь заданий (общая для server и worker).
	QueueAvatarJobs = "gophprofile.avatars"
)

// EnsureAvatarsTopology объявляет exchange topic, очередь и привязки routing key.
func EnsureAvatarsTopology(ch *amqp.Channel) (amqp.Queue, error) {
	// Объявляем exchange topic
	if err := ch.ExchangeDeclare(
		ExchangeAvatars,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return amqp.Queue{}, err
	}

	// Объявляем очередь
	q, err := ch.QueueDeclare(
		QueueAvatarJobs,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return amqp.Queue{}, err
	}

	// Привязываем очередь к routing key
	for _, rk := range []string{RoutingKeyAvatarProcess, RoutingKeyAvatarDelete} {
		if err := ch.QueueBind(q.Name, rk, ExchangeAvatars, false, nil); err != nil {
			return amqp.Queue{}, err
		}
	}

	return q, nil
}
