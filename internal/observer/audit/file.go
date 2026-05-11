package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/coolycow/gophprofile/internal/model"
)

// FileReceiver записывает события аудита в файл (каждое событие — новая строка).
type FileReceiver struct {
	file *os.File
	mu   sync.Mutex
}

// NewFileReceiver открывает файл для дописывания и держит его открытым до Close.
func NewFileReceiver(path string) (*FileReceiver, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open audit file %s: %w", path, err)
	}
	return &FileReceiver{file: f}, nil
}

// Send добавляет событие в конец файла в виде одной строки JSON.
func (f *FileReceiver) Send(event *model.Audit) error {
	// Преобразуем событие в JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	// Добавляем символ новой строки
	data = append(data, '\n')

	// Записываем событие в файл
	f.mu.Lock()
	_, err = f.file.Write(data)
	f.mu.Unlock()

	// Если ошибка при записи в файл, возвращаем ошибку
	if err != nil {
		return fmt.Errorf("write audit event to file: %w", err)
	}

	return nil
}

// Close сбрасывает буферы и закрывает файл.
func (f *FileReceiver) Close() error {
	// Блокируем доступ к файлу для синхронизации
	f.mu.Lock()
	defer f.mu.Unlock()

	// Если файл не открыт, возвращаем nil
	if f.file == nil {
		return nil
	}

	// Закрываем файл
	err := f.file.Close()
	f.file = nil

	return err
}
