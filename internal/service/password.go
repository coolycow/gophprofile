package service

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/coolycow/gophprofile/internal/config"
	"github.com/go-playground/validator/v10"
)

// hashPassword хэширует пароль и возвращает строку
func hashPassword(password string) (string, error) {
	// Чем больше cost тем больше итераций хеширования.
	// При cost=14 выполняется 16384 итераций, а при cost=10 - всего 1024 итераций.
	// При cost=14 время входа 781.1083ms, при cost=10 - 49.4006ms.
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

// validatePassword валидирует пароль с учетом конфигурации и валидатора
func validatePassword(password string, cfg *config.ConfigServer, validator *validator.Validate) error {
	// Валидация пароля
	return validator.Var(password, fmt.Sprintf("required,alphanum,min=%d,max=%d", cfg.MinPasswordLength, cfg.MaxPasswordLength))
}
