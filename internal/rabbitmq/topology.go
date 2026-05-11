package rabbitmq

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// QueueAvatarJobs — durable-очередь заданий на обработку аватаров (общая для сервера и воркера).
	QueueAvatarJobs = "gophprofile.avatars"
)

// EnsureAvatarJobsQueue объявляет очередь идемпотентно при совместимых аргументах.
func EnsureAvatarJobsQueue(ch *amqp.Channel) (amqp.Queue, error) {
	return ch.QueueDeclare(
		QueueAvatarJobs,
		true,
		false,
		false,
		false,
		nil,
	)
}
