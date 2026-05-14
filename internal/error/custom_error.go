// Package error определяет пользовательские типы ошибок для HTTP-ответов.
package error

// CustomError — ошибка с HTTP-кодом для возврата клиенту.
type CustomError struct {
	Message    string
	Details    string // необязательное поле для JSON (например структурированное описание)
	StatusCode int
	// Meta — дополнительные поля JSON (например max_size для 413).
	Meta map[string]any
}

func (e CustomError) Error() string {
	return e.Message
}
