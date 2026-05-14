package service

import (
	"crypto/rand"
)

// generateRandom генерация случайных байт заданного размера
func generateRandom(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)

	if err != nil {
		return nil, err
	}

	return b, nil
}
