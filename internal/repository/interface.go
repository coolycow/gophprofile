// Package repository определяет интерфейсы и реализации хранилища URL.
package repository

import (
	"context"
	"time"

	"github.com/coolycow/gophprofile/internal/model"
)

// GophProfileRepository — интерфейс хранилища: сохранение/получение аватаров, пользователи, пинг, миграции.
type GophProfileRepository interface {
	// Методы для работы с базой данных
	Close() error                   // закрывает соединение с базой данных
	Ping(ctx context.Context) error // проверяет доступность базы данных
	RunMigrations() error           // выполняет миграции

	////////////////////////////////////////////////////////////// МЕТОДЫ ДЛЯ РАБОТЫ С ПОЛЬЗОВАТЕЛЯМИ //////////////////////////////////////////////////////////////
	GetUserByID(ctx context.Context, userID string) (*model.User, error)   // получает пользователя по его ID
	GetUserByEmail(ctx context.Context, email string) (*model.User, error) // получает пользователя по его email
	CreateUser(ctx context.Context, user *model.User) (*model.User, error) // создает нового пользователя
	UpdateUser(ctx context.Context, userID string, user *model.User) error // обновляет пользователя
	SoftDeleteUser(ctx context.Context, userID string) error               // мягко удаляет пользователя
	HardDeleteUser(ctx context.Context, userID string) error               // полностью удаляет пользователя
	GetUsersCount(ctx context.Context) int                                 // возвращает количество пользователей

	////////////////////////////////////////////////////////////// МЕТОДЫ ДЛЯ REFRESH-ТОКЕНОВ //////////////////////////////////////////////////////////////
	CreateRefreshToken(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time) (*model.RefreshToken, error) // создаёт запись refresh-токена
	DeleteRefreshToken(ctx context.Context, id string) error                                                                   // удаляет запись (ротация / отзыв)
	FindValidRefreshTokenByHash(ctx context.Context, tokenHash []byte, now time.Time) (*model.RefreshToken, error)             // находит неистёкшую запись по хэшу

	////////////////////////////////////////////////////////////// МЕТОДЫ ДЛЯ РАБОТЫ С АВАТАРАМИ //////////////////////////////////////////////////////////////
	UploadAvatar(ctx context.Context, userID string, fileName string, mimeType string, sizeBytes int64,
		s3Key string, thumbnailS3Keys string, uploadStatus string, processingStatus string) (*model.Avatar, error) // загружает аватарку
	GetAvatarByID(ctx context.Context, avatarID string) (*model.Avatar, error)   // получает аватарку по ID
	GetAvatarByUserID(ctx context.Context, userID string) (*model.Avatar, error) // получает аватарку по ID пользователя
	DeleteAvatarByID(ctx context.Context, avatarID string) error                 // удаляет аватарку по ID
	DeleteAvatarByUserID(ctx context.Context, userID string) error               // удаляет аватарку по ID пользователя
	GetUserAvatars(ctx context.Context, userID string) ([]*model.Avatar, error)  // получает список аватарок пользователя
}
