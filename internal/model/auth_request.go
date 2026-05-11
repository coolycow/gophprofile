package model

// LoginRequest модель авторизации
// Валидация в сервисе user.go
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterRequest модель регистрации
// Валидация в сервисе user.go
type UserRegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UpdateRequest модель обновления пользователя
// Валидация в сервисе user.go
type UserUpdateRequest struct {
	Email       string `json:"email,omitempty"`
	OldPassword string `json:"old_password,omitempty"`
	NewPassword string `json:"new_password,omitempty"`
}

// RefreshTokenRequest тело запроса обновления пары токенов
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}
