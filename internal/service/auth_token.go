package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	profileError "github.com/coolycow/gophprofile/internal/error"
	"github.com/coolycow/gophprofile/internal/model"
)

// IssueAuthTokens выдаёт JWT access и непрозрачный refresh; refresh сохраняется в БД по SHA-256 от строки.
func (s *userService) IssueAuthTokens(ctx context.Context, user *model.User) (accessToken, refreshToken string, accessExpiresAt time.Time, err error) {
	// Подписываем токен доступа
	accessToken, accessExpiresAt, err = s.signAccessToken(user.ID)
	if err != nil {
		return "", "", time.Time{}, err
	}

	// Генерируем случайные байты для refresh-токена
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", time.Time{}, err
	}

	// Преобразуем байты в строку
	refreshToken = hex.EncodeToString(raw)

	// Преобразуем строку в хэш
	sum := sha256.Sum256([]byte(refreshToken))

	// Получаем срок действия refresh-токена
	expiresAt := time.Now().Add(s.refreshTTL())

	// Создаём запись refresh-токена в БД
	_, err = s.repo.CreateRefreshToken(ctx, user.ID, sum[:], expiresAt)
	if err != nil {
		return "", "", time.Time{}, err
	}

	// Возвращаем токены доступа и срок действия
	return accessToken, refreshToken, accessExpiresAt, nil
}

// ExchangeRefreshToken проверяет refresh в БД, отзывает запись и выдаёт новую пару токенов.
func (s *userService) ExchangeRefreshToken(ctx context.Context, refreshPlain string) (*model.User, string, string, time.Time, error) {
	// Очищаем refresh-токен от пробелов
	refreshPlain = strings.TrimSpace(refreshPlain)

	// Если refresh-токен пустой, возвращаем ошибку
	if refreshPlain == "" {
		return nil, "", "", time.Time{}, profileError.CustomError{
			Message:    "refresh token required",
			StatusCode: http.StatusUnauthorized,
		}
	}

	// Преобразуем refresh-токен в хэш
	sum := sha256.Sum256([]byte(refreshPlain))

	// Ищем запись refresh-токена в БД
	row, err := s.repo.FindValidRefreshTokenByHash(ctx, sum[:], time.Now())

	// Если запись не найдена, возвращаем ошибку
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", "", time.Time{}, profileError.CustomError{
				Message:    "invalid refresh token",
				StatusCode: http.StatusUnauthorized,
			}
		}
		return nil, "", "", time.Time{}, err
	}

	// Получаем пользователя по ID
	u, err := s.GetUserByID(ctx, row.UserID)
	if err != nil {
		return nil, "", "", time.Time{}, err
	}
	if u == nil {
		return nil, "", "", time.Time{}, profileError.CustomError{
			Message:    "user not found",
			StatusCode: http.StatusUnauthorized,
		}
	}

	// Если пользователь мягко удалён, возвращаем ошибку
	if u.DeletedAt != nil && !u.DeletedAt.IsZero() {
		return nil, "", "", time.Time{}, profileError.CustomError{
			Message:    "user deleted",
			StatusCode: http.StatusUnauthorized,
		}
	}

	// Выдаём новые токены доступа
	access, newRefresh, exp, err := s.IssueAuthTokens(ctx, u)
	if err != nil {
		return nil, "", "", time.Time{}, err
	}

	// Удаляем запись refresh-токена из БД
	if err := s.repo.DeleteRefreshToken(ctx, row.ID); err != nil {
		return nil, "", "", time.Time{}, err
	}

	return u, access, newRefresh, exp, nil
}

// GetUserIDFromAuthToken проверяет JWT access (metadata authorization / Bearer).
func (s *userService) GetUserIDFromAuthToken(token string) (string, error) {
	// Очищаем токен от пробелов
	token = strings.TrimSpace(token)

	// Если токен пустой, возвращаем ошибку
	if len(token) > 6 && strings.EqualFold(token[:7], "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	if token == "" {
		return "", errors.New("missing token")
	}

	// Парсим токен
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.SecretKey), nil
	})

	// Если ошибка при парсинге токена, возвращаем ошибку
	if err != nil {
		return "", err
	}

	// Если subject пустой, возвращаем ошибку
	if claims.Subject == "" {
		return "", errors.New("invalid token claims")
	}

	// Возвращаем userID из токена
	return claims.Subject, nil
}

// signAccessToken подписывает токен доступа
func (s *userService) signAccessToken(userID string) (string, time.Time, error) {
	// Получаем текущую дату и время
	now := time.Now()

	// Получаем срок действия токена доступа
	exp := now.Add(s.accessTTL())

	// Создаём claims для токена доступа
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(exp),
		IssuedAt:  jwt.NewNumericDate(now),
	}

	// Создаём токен с claims
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен
	signed, err := tok.SignedString([]byte(s.cfg.SecretKey))

	// Если ошибка при подписывании токена, возвращаем ошибку
	if err != nil {
		return "", time.Time{}, err
	}

	// Возвращаем подписанный токен и срок действия
	return signed, exp, nil
}

// accessTTL возвращает срок действия токена доступа
func (s *userService) accessTTL() time.Duration {
	return time.Duration(s.cfg.AccessTokenTTLMinutes) * time.Minute
}

// refreshTTL возвращает срок действия refresh-токена
func (s *userService) refreshTTL() time.Duration {
	return time.Duration(s.cfg.RefreshTokenTTLHours) * time.Hour
}
