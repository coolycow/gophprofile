// Package audit реализует отправку событий аудита в файл и по HTTP.
package audit

import (
	"errors"
	"sync"
	"time"

	"github.com/coolycow/gophprofile/internal/logger"
	"github.com/coolycow/gophprofile/internal/model"
	"go.uber.org/zap"
)

// Константы типа действия в событии аудита.
const (
	ActionRegister     = "register"      // регистрация пользователя
	ActionLogin        = "login"         // вход по email/паролю
	ActionRefreshToken = "refresh_token" // обновление пары токенов по refresh

	ActionCreateSecret = "create_secret" // создание секрета
	ActionUpdateSecret = "update_secret" // обновление секрета
	ActionDeleteSecret = "delete_secret" // удаление секрета

	ActionCreateSecretVersion = "create_secret_version" // создание версии секрета
	ActionUpdateSecretVersion = "update_secret_version" // обновление версии секрета
	ActionDeleteSecretVersion = "delete_secret_version" // удаление версии секрета

	ActionCreateAttachment = "create_attachment" // создание приложения
	ActionUpdateAttachment = "update_attachment" // обновление приложения
	ActionDeleteAttachment = "delete_attachment" // удаление приложения
)

// Receiver — приёмник событий аудита (файл, HTTP и т.д.).
type Receiver interface {
	Send(event *model.Audit) error
}

// Notifier рассылает события аудита всем зарегистрированным приёмникам.
// Потокобезопасен, поддерживает динамическое добавление и удаление приёмников.
type Notifier struct {
	mu        sync.RWMutex
	receivers []Receiver
}

// Notify отправляет событие во все приёмники (ошибки приёмников не блокируют рассылку).
func (n *Notifier) Notify(event *model.Audit) {
	n.mu.RLock()
	receivers := make([]Receiver, len(n.receivers))
	copy(receivers, n.receivers)
	n.mu.RUnlock()

	for _, r := range receivers {
		if err := r.Send(event); err != nil {
			logger.Log.Warn("audit receiver error", zap.Error(err))
		}
	}
}

// AddReceiver добавляет новый приёмник.
func (n *Notifier) AddReceiver(r Receiver) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.receivers = append(n.receivers, r)
}

// RemoveReceiver удаляет существующий приёмник (первое совпадение по ссылке).
func (n *Notifier) RemoveReceiver(r Receiver) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for i, recv := range n.receivers {
		if recv == r {
			n.receivers = append(n.receivers[:i], n.receivers[i+1:]...)
			return
		}
	}
}

// NewEvent создаёт событие аудита с текущим временем. Неиспользуемые идентификаторы ресурсов передавайте пустыми строками.
func NewEvent(action, userID, avatarID, processingStatus string) *model.Audit {
	return &model.Audit{
		TS:               int(time.Now().Unix()),
		Action:           action,
		UserID:           userID,
		AvatarID:         avatarID,
		ProcessingStatus: processingStatus,
	}
}

// NewNotifier создаёт Notifier: при auditFile != "" — запись в файл, при auditURL != "" — отправка на URL.
// Если оба параметра не пустые, то запись в файл и отправка на URL.
func NewNotifier(auditFile, auditURL string) (*Notifier, error) {
	n := &Notifier{}

	if auditFile != "" {
		fr, err := NewFileReceiver(auditFile)
		if err != nil {
			return nil, err
		}
		n.AddReceiver(fr)
	}

	if auditURL != "" {
		n.AddReceiver(NewURLReceiver(auditURL, newAuditHTTPClient()))
	}

	return n, nil
}

// Close закрывает приёмники, которым это нужно (например файл аудита).
func (n *Notifier) Close() error {
	n.mu.Lock()
	receivers := append([]Receiver(nil), n.receivers...)
	n.mu.Unlock()

	var errs []error
	for _, r := range receivers {
		if c, ok := r.(interface{ Close() error }); ok {
			if err := c.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}
