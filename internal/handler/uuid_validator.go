package handler

import "github.com/google/uuid"

// ValidateUUID проверяет, является ли строка UUID.
func ValidateUUID(uid string) error {
	if _, err := uuid.Parse(uid); err != nil {
		return err
	}
	return nil
}
