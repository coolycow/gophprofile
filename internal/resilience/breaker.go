package resilience

import (
	"errors"
	"time"

	"github.com/sony/gobreaker"
)

// ErrCircuitOpen возвращается, когда circuit breaker открыт.
var ErrCircuitOpen = errors.New("service temporarily unavailable: circuit breaker open")

// Breakers — набор circuit breaker'ов для внешних зависимостей.
// Создаётся при старте приложения и передаётся в компоненты явно (без package-level globals).
type Breakers struct {
	Minio    *gobreaker.CircuitBreaker
	RabbitMQ *gobreaker.CircuitBreaker
	Postgres *gobreaker.CircuitBreaker
}

// NewBreakers инициализирует breaker'ы для MinIO, RabbitMQ и PostgreSQL.
func NewBreakers() *Breakers {
	return &Breakers{
		Minio:    NewBreaker("minio"),
		RabbitMQ: NewBreaker("rabbitmq"),
		Postgres: NewBreaker("postgres"),
	}
}

// NewBreaker создаёт circuit breaker с разумными defaults для внешних зависимостей.
func NewBreaker(name string) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	})
}

// Execute выполняет fn через circuit breaker и нормализует ошибку open state.
func Execute[T any](cb *gobreaker.CircuitBreaker, fn func() (T, error)) (T, error) {
	var zero T
	if cb == nil {
		return fn()
	}

	v, err := cb.Execute(func() (any, error) {
		return fn()
	})
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return zero, ErrCircuitOpen
		}
		return zero, err
	}
	if v == nil {
		return zero, nil
	}
	result, ok := v.(T)
	if !ok {
		return zero, err
	}
	return result, nil
}

// ExecuteVoid — Execute без возвращаемого значения.
func ExecuteVoid(cb *gobreaker.CircuitBreaker, fn func() error) error {
	_, err := Execute(cb, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}
