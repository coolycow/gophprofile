package service

// OrNA возвращает строку "N/A" если строка пустая
func OrNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}
