package audit

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coolycow/gophprofile/internal/model"
	"github.com/hashicorp/go-retryablehttp"
)

// URLReceiver отправляет события аудита на удалённый сервер методом POST.
type URLReceiver struct {
	url    string
	client *retryablehttp.Client
}

// newAuditHTTPClient возвращает HTTP-клиент с ретраями для отправки аудита.
func newAuditHTTPClient() *retryablehttp.Client {
	return retryablehttp.NewClient()
}

// NewURLReceiver создаёт приёмник на удалённый URL; client задаёт транспорт (ретраи и т.п.).
func NewURLReceiver(auditURL string, client *retryablehttp.Client) *URLReceiver {
	return &URLReceiver{url: auditURL, client: client}
}

// Send отправляет событие POST-запросом с JSON-телом (с автоматическими ретраями).
func (u *URLReceiver) Send(event *model.Audit) error {
	// Преобразуем событие в JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	// Создаём запрос
	req, err := retryablehttp.NewRequest(http.MethodPost, u.url, data)
	if err != nil {
		return fmt.Errorf("create audit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Отправляем запрос
	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("send audit event to %s: %w", u.url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Если статус ответа не в диапазоне 200-299, возвращаем ошибку
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("audit endpoint returned status %d", resp.StatusCode)
	}

	return nil
}
