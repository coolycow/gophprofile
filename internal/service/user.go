package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"

	"github.com/coolycow/gophprofile/internal/config"
	profileError "github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/model"
	"github.com/coolycow/gophprofile/internal/repository"
)

// UserService Сервис для работы с пользователями
type UserService interface {
	// Методы получения пользователя
	GetUserByID(ctx context.Context, userID string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetUserByEmailAndPassword(ctx context.Context, email string, password string) (*model.User, error)

	// Методы для работы с пользователем
	CreateUser(ctx context.Context, request model.UserRegisterRequest) (*model.User, error)
	UpdateUser(ctx context.Context, userID string, request model.UserUpdateRequest) error

	// Методы для работы с мягким и полным удалением пользователя
	SoftDeleteUser(ctx context.Context, userID string) error
	HardDeleteUser(ctx context.Context, userID string) error

	// IssueAuthTokens выдаёт JWT access и непрозрачный refresh (refresh хранится в БД по хэшу).
	IssueAuthTokens(ctx context.Context, user *model.User) (accessToken, refreshToken string, accessExpiresAt time.Time, err error)

	// ExchangeRefreshToken проверяет refresh в БД и выдаёт новую пару токенов.
	ExchangeRefreshToken(ctx context.Context, refreshToken string) (*model.User, string, string, time.Time, error)

	// GetUserIDFromAuthToken проверяет JWT access (metadata authorization / Bearer).
	GetUserIDFromAuthToken(token string) (string, error)
}

// Реализация сервисного слоя
type userService struct {
	repo      repository.GophProfileRepository
	cfg       *config.ConfigServer
	validator *validator.Validate
}

// NewUserService инициализация сервиса
func NewUserService(cfg *config.ConfigServer, repo repository.GophProfileRepository) UserService {
	return &userService{
		repo:      repo,
		cfg:       cfg,
		validator: validator.New(),
	}
}

// GetUserByID возвращает пользователя по его ID
func (s *userService) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

// GetUserByEmail возвращает пользователя по его email
func (s *userService) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	// Валидация email
	if err := s.validator.Var(email, "required,email"); err != nil {
		return nil, profileError.CustomError{
			Message:    fmt.Sprintf("email validation failed: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	// Получаем пользователя по email
	return s.repo.GetUserByEmail(ctx, email)
}

// GetUserByEmailAndPassword возвращает пользователя по его email и паролю
func (s *userService) GetUserByEmailAndPassword(ctx context.Context, email string, password string) (*model.User, error) {
	// Получаем пользователя по email (email будет валидирован в GetUserByEmail)
	user, err := s.GetUserByEmail(ctx, email)

	// Если ошибка, возвращаем ошибку (уже тип profileError.CustomError)
	if err != nil {
		return nil, err
	}

	// Если пользователь не найден, возвращаем ошибку
	if user == nil {
		return nil, profileError.CustomError{
			Message:    "user not found",
			StatusCode: http.StatusUnauthorized,
		}
	}

	// Сравниваем пароль с хешем в базе данных
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))

	// Если пароль не совпадает, возвращаем ошибку
	if err != nil {
		return nil, profileError.CustomError{
			Message:    err.Error(),
			StatusCode: http.StatusUnauthorized,
		}
	}

	return user, nil
}

// CreateUser создаёт нового пользователя
func (s *userService) CreateUser(ctx context.Context, request model.UserRegisterRequest) (*model.User, error) {
	// Валидация почты
	if err := s.validator.Var(request.Email, "required,email"); err != nil {
		return nil, profileError.CustomError{
			Message:    fmt.Sprintf("email validation failed: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	// Валидация пароля
	if err := validatePassword(request.Password, s.cfg, s.validator); err != nil {
		return nil, profileError.CustomError{
			Message:    fmt.Sprintf("password validation failed: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	// Проверяем, существует ли пользователь с такой email
	existingUser, err := s.repo.GetUserByEmail(ctx, request.Email)

	// Если ошибка, возвращаем ошибку, если она не является ошибкой отсутствия пользователя
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, profileError.CustomError{
			Message:    err.Error(),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Если пользователь с такой email уже существует, возвращаем ошибку
	if existingUser != nil {
		return nil, profileError.CustomError{
			Message:    "user with this email already exists",
			StatusCode: http.StatusConflict,
		}
	}

	// Хешируем пароль
	hashedPassword, err := hashPassword(request.Password)

	// Если ошибка хеширования пароля, возвращаем ошибку
	if err != nil {
		return nil, profileError.CustomError{
			Message:    err.Error(),
			StatusCode: http.StatusInternalServerError,
		}
	}

	now := time.Now()

	// Создаём пользователя
	return s.repo.CreateUser(ctx, &model.User{
		Email:     request.Email,
		Password:  hashedPassword,
		CreatedAt: &now,
		UpdatedAt: &now,
	})
}

// UpdateUser обновляет пользователя
func (s *userService) UpdateUser(ctx context.Context, userID string, request model.UserUpdateRequest) error {
	// Проверяем, существует ли пользователь с таким ID
	user, err := s.repo.GetUserByID(ctx, userID)

	if err != nil {
		return profileError.CustomError{
			Message:    err.Error(),
			StatusCode: http.StatusInternalServerError,
		}
	}

	if user == nil {
		return profileError.CustomError{
			Message:    "user not found",
			StatusCode: http.StatusNotFound,
		}
	}

	// Обновляем пароль если он изменился
	if request.OldPassword != "" && request.NewPassword != "" {
		// Сравниваем старый пароль с хешем в базе данных, если он не совпадает, возвращаем ошибку
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.OldPassword))
		if err != nil {
			return profileError.CustomError{
				Message:    "old password is incorrect",
				StatusCode: http.StatusUnauthorized,
			}
		}

		// Валидация нового пароля
		if err := validatePassword(request.NewPassword, s.cfg, s.validator); err != nil {
			return profileError.CustomError{
				Message:    fmt.Sprintf("password validation failed: %s", err),
				StatusCode: http.StatusBadRequest,
			}
		}

		// Хешируем новый пароль
		hashedPassword, err := hashPassword(request.NewPassword)
		if err != nil {
			return profileError.CustomError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			}
		}

		// Обновляем пароль
		user.Password = hashedPassword
	}

	// Валидация email если email изменился
	if request.Email != "" && request.Email != user.Email {
		if err := s.validator.Var(request.Email, "email"); err != nil {
			return profileError.CustomError{
				Message:    fmt.Sprintf("email validation failed: %s", err),
				StatusCode: http.StatusBadRequest,
			}
		}

		// Проверяем, существует ли пользователь с такой email
		existingUser, err := s.repo.GetUserByEmail(ctx, request.Email)

		if err != nil {
			return profileError.CustomError{
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			}
		}

		// Если пользователь с такой email уже существует, возвращаем ошибку
		if existingUser != nil {
			return profileError.CustomError{
				Message:    "user with this email already exists",
				StatusCode: http.StatusConflict,
			}
		}

		// Обновляем email
		user.Email = request.Email
	}

	now := time.Now()

	// Обновляем пользователя
	return s.repo.UpdateUser(ctx, userID, &model.User{
		Email:     user.Email,    // актуальный email
		Password:  user.Password, // Пароль может быть изменен или не изменен
		UpdatedAt: &now,
	})
}

// SoftDeleteUser мягко удаляет пользователя
func (s *userService) SoftDeleteUser(ctx context.Context, userID string) error {
	return s.repo.SoftDeleteUser(ctx, userID)
}

// HardDeleteUser полностью удаляет пользователя
func (s *userService) HardDeleteUser(ctx context.Context, userID string) error {
	return s.repo.HardDeleteUser(ctx, userID)
}
