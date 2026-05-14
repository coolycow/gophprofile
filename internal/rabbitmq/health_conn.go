package rabbitmq

import amqp "github.com/rabbitmq/amqp091-go"

// HealthConn обёртка над AMQP-соединением для /health (IsClosed + число consumer очереди).
type HealthConn struct {
	*amqp.Connection
}

// NewHealthConn возвращает обёртку; при c == nil вернёт nil.
func NewHealthConn(c *amqp.Connection) *HealthConn {
	if c == nil {
		return nil
	}
	return &HealthConn{Connection: c}
}

// QueueConsumerCount пассивно объявляет очередь и возвращает число подписчиков.
func (h *HealthConn) QueueConsumerCount(queue string) (int, error) {
	// Проверяем, не nil ли соединение
	if h == nil || h.Connection == nil {
		return 0, nil
	}

	// Получаем канал
	ch, err := h.Connection.Channel()
	if err != nil {
		return 0, err
	}
	defer ch.Close()

	// Объявляем очередь пассивно
	q, err := ch.QueueDeclarePassive(queue, true, false, false, false, nil)
	if err != nil {
		return 0, err
	}

	// Возвращаем число подписчиков
	return q.Consumers, nil
}
