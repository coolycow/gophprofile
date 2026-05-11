// Package error определяет пользовательские типы ошибок для HTTP-ответов.
package error

// CustomError — ошибка с HTTP-кодом для возврата клиенту.
type CustomError struct {
	Message    string
	Details    string // необязательное поле для JSON (например структурированное описание)
	StatusCode int
}

func (e CustomError) Error() string {
	return e.Message
}
