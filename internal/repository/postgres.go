package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coolycow/gophprofile/internal/logger"
	"github.com/coolycow/gophprofile/internal/model"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// PostgresRepository представляет репозиторий для хранения URL
type PostgresRepository struct {
	db *sql.DB
}

// checkTableExists проверяет, существует ли таблица
func (r *PostgresRepository) checkTableExists(tableName string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)`, tableName).Scan(&exists)
	return exists, err
}

// findMigrationsDir возвращает абсолютный путь к каталогу SQL-миграций.
// Приоритет: MIGRATIONS_PATH → ./migrations относительно cwd → подъём к каталогу с go.mod.
func findMigrationsDir() (string, error) {
	if p := strings.TrimSpace(os.Getenv("MIGRATIONS_PATH")); p != "" {
		abs, err := filepath.Abs(p)
		if err != nil {
			return "", fmt.Errorf("MIGRATIONS_PATH: %w", err)
		}
		fi, err := os.Stat(abs)
		if err != nil {
			return "", fmt.Errorf("MIGRATIONS_PATH %q: %w", p, err)
		}
		if !fi.IsDir() {
			return "", fmt.Errorf("MIGRATIONS_PATH %q is not a directory", p)
		}
		return abs, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	candidate := filepath.Join(wd, "migrations")
	if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
		return filepath.Abs(candidate)
	}

	projectRoot := wd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			p := filepath.Join(projectRoot, "migrations")
			if fi, err := os.Stat(p); err == nil && fi.IsDir() {
				return filepath.Abs(p)
			}
			return "", fmt.Errorf("migrations directory not found under project root %q", projectRoot)
		}

		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			return "", fmt.Errorf("project root (go.mod) not found; set MIGRATIONS_PATH or run from the repository root")
		}
		projectRoot = parent
	}
}

// RunMigrations выполняет миграции
func (r *PostgresRepository) RunMigrations() error {
	logger.Log.Info("Running migrations")

	// Создаем экземпляр драйвера для PostgreSQL
	driver, err := pgx.WithInstance(r.db, &pgx.Config{})
	if err != nil {
		return err
	}

	migrationsPath, err := findMigrationsDir()
	if err != nil {
		return err
	}

	migrationsURL := "file://" + filepath.ToSlash(migrationsPath)

	// Указываем путь к директории с миграциями
	m, err := migrate.NewWithDatabaseInstance(migrationsURL, "pgx", driver)
	if err != nil {
		return err
	}

	// Применяем миграции
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

// NewPostgresRepository создает новый экземпляр URLRepository
func NewPostgresRepository(DSN string) (*PostgresRepository, error) {
	// Открываем соединение с базой данных
	db, err := sql.Open("pgx", DSN)

	// Ошибка открытия соединения
	if err != nil {
		return nil, err
	}

	// Проверяем, доступна ли база данных
	if err = db.Ping(); err != nil {
		closeErr := db.Close()
		if closeErr != nil {
			log.Printf("Error closing database: %v", closeErr)
		}
		return nil, err
	}

	repo := &PostgresRepository{db: db}

	// Проверяем, существует ли таблица users
	tableExists, err := repo.checkTableExists("users")
	if err != nil {
		return nil, err
	}

	// Если таблица users не существует, выполняем миграции, т.к. это явно первый запуск приложения на сервере
	if !tableExists {
		err = repo.RunMigrations()

		if err != nil {
			return nil, err
		}
	}

	return repo, nil
}

// //////////////////////////////////////////////////////////// МЕТОДЫ ДЛЯ РАБОТЫ С БАЗОЙ ДАННЫХ //////////////////////////////////////////////////////////////
// Close закрывает хранилище
func (r *PostgresRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// Ping проверяет доступность хранилища
func (r *PostgresRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

////////////////////////////////////////////////////////////// МЕТОДЫ ДЛЯ РАБОТЫ С ПОЛЬЗОВАТЕЛЯМИ //////////////////////////////////////////////////////////////

// GetUserByID получает пользователя по его ID
func (r *PostgresRepository) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	row := r.db.QueryRowContext(ctx, "select id, email, password, created_at, updated_at, deleted_at from users where id = $1", userID)

	var user model.User
	err := row.Scan(&user.ID, &user.Email, &user.Password,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByEmail получает пользователя по его email
func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	row := r.db.QueryRowContext(ctx, "select id, email, password, created_at, updated_at, deleted_at from users where email = $1", email)

	var user model.User
	err := row.Scan(&user.ID, &user.Email, &user.Password,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// CreateUser создает нового пользователя
func (r *PostgresRepository) CreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	row := r.db.QueryRowContext(ctx, `insert into users (email, password) 
	values ($1, $2) returning id, email, password, created_at, updated_at, deleted_at`,
		user.Email, user.Password)

	var newUser model.User
	err := row.Scan(&newUser.ID, &newUser.Email, &newUser.Password,
		&newUser.CreatedAt, &newUser.UpdatedAt, &newUser.DeletedAt)

	if err != nil {
		return nil, err
	}

	return &newUser, nil
}

// UpdateUser обновляет пользователя
func (r *PostgresRepository) UpdateUser(ctx context.Context, userID string, user *model.User) error {
	row := r.db.QueryRowContext(ctx, `update users 
	set email = $1, password = $2, updated_at = $3, deleted_at = $4 where id = $5
	returning id, email, password, created_at, updated_at, deleted_at`,
		user.Email, user.Password, user.UpdatedAt, user.DeletedAt, userID)

	err := row.Scan(&user.ID, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)

	if err != nil {
		return err
	}

	return nil
}

// SoftDeleteUser мягко удаляет пользователя
func (r *PostgresRepository) SoftDeleteUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, "update users set deleted_at = $1 where id = $2", time.Now(), userID)

	if err != nil {
		return err
	}

	return nil
}

// HardDeleteUser полностью удаляет пользователя
func (r *PostgresRepository) HardDeleteUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, "delete from users where id = $1", userID)

	if err != nil {
		return err
	}

	return nil
}

// GetUsersCount возвращает число записей в таблице users.
func (r *PostgresRepository) GetUsersCount(ctx context.Context) int {
	row := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users")

	var count int64

	if err := row.Scan(&count); err != nil {
		return 0
	}

	return int(count)
}

// //////////////////////////////////////////////////////////// МЕТОДЫ ДЛЯ REFRESH-ТОКЕНОВ //////////////////////////////////////////////////////////////
// CreateRefreshToken создаёт запись refresh-токена (в БД только SHA-256 от переданной клиенту строки).
func (r *PostgresRepository) CreateRefreshToken(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time) (*model.RefreshToken, error) {
	row := r.db.QueryRowContext(ctx,
		`insert into refresh_tokens (user_id, token_hash, expires_at) values ($1, $2, $3)
		returning id, user_id, token_hash, expires_at, created_at`,
		userID, tokenHash, expiresAt)

	var t model.RefreshToken
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &t, nil
}

// DeleteRefreshToken удаляет запись refresh-токена по первичному ключу.
func (r *PostgresRepository) DeleteRefreshToken(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `delete from refresh_tokens where id = $1`, id)
	return err
}

// FindValidRefreshTokenByHash возвращает неистёкшую запись с данным хэшем или sql.ErrNoRows.
func (r *PostgresRepository) FindValidRefreshTokenByHash(ctx context.Context, tokenHash []byte, now time.Time) (*model.RefreshToken, error) {
	row := r.db.QueryRowContext(ctx,
		`select id, user_id, token_hash, expires_at, created_at from refresh_tokens
		where token_hash = $1 and expires_at > $2`,
		tokenHash, now)

	var t model.RefreshToken
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

////////////////////////////////////////////////////////////// МЕТОДЫ ДЛЯ РАБОТЫ С АВАТАРАМИ //////////////////////////////////////////////////////////////

// UploadAvatar загружает аватарку (если есть текущая аватарка, то мягко удаляет её)
func (r *PostgresRepository) UploadAvatar(ctx context.Context, userID string, fileName string, mimeType string, sizeBytes int64,
	s3Key string, thumbnailS3Keys string, uploadStatus string, processingStatus string, currentAvatarID string) (*model.Avatar, error) {
	// Начинаем транзакцию
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Мягко удаляем текущую аватарку (если есть)
	if currentAvatarID != "" {
		_, err = tx.ExecContext(ctx, "update avatars set deleted_at = $1 where id = $2", time.Now(), currentAvatarID)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Сохраняем новую аватарку в базу данных
	row := tx.QueryRowContext(ctx, `insert into avatars (user_id, file_name, mime_type, size_bytes, 
	s3_key, thumbnail_s3_keys, upload_status, processing_status)
	values ($1, $2, $3, $4, $5, $6, $7, $8) 
	returning id, user_id, file_name, mime_type, size_bytes, s3_key, thumbnail_s3_keys, upload_status, processing_status, created_at, updated_at, deleted_at`,
		userID, fileName, mimeType, sizeBytes, s3Key, thumbnailS3Keys, uploadStatus, processingStatus)

	// Сканируем результат
	var a model.Avatar
	err = row.Scan(&a.ID, &a.UserID, &a.FileName, &a.MimeType, &a.SizeBytes,
		&a.S3Key, &a.ThumbnailS3Keys, &a.UploadStatus, &a.ProcessingStatus,
		&a.CreatedAt, &a.UpdatedAt, &a.DeletedAt)

	// Если ошибка, откатываем транзакцию
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Фиксируем транзакцию
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &a, nil
}

// GetAvatarByID получает аватарку по ID (только не удаленные)
func (r *PostgresRepository) GetAvatarByID(ctx context.Context, avatarID string) (*model.Avatar, error) {
	row := r.db.QueryRowContext(ctx, `select id, user_id, file_name, mime_type, size_bytes, 
	s3_key, thumbnail_s3_keys, upload_status, processing_status, 
	created_at, updated_at, deleted_at from avatars where id = $1 and deleted_at is null`, avatarID)

	var a model.Avatar
	err := row.Scan(&a.ID, &a.UserID, &a.FileName, &a.MimeType, &a.SizeBytes,
		&a.S3Key, &a.ThumbnailS3Keys, &a.UploadStatus, &a.ProcessingStatus,
		&a.CreatedAt, &a.UpdatedAt, &a.DeletedAt)

	if err != nil {
		return nil, err
	}

	return &a, nil
}

// GetAvatarByUserID получает аватарку по ID пользователя (только не удаленные)
func (r *PostgresRepository) GetAvatarByUserID(ctx context.Context, userID string) (*model.Avatar, error) {
	row := r.db.QueryRowContext(ctx, `select id, user_id, file_name, mime_type, size_bytes, 
	s3_key, thumbnail_s3_keys, upload_status, processing_status, 
	created_at, updated_at, deleted_at from avatars where user_id = $1 and deleted_at is null`, userID)

	var a model.Avatar
	err := row.Scan(&a.ID, &a.UserID, &a.FileName, &a.MimeType, &a.SizeBytes,
		&a.S3Key, &a.ThumbnailS3Keys, &a.UploadStatus, &a.ProcessingStatus,
		&a.CreatedAt, &a.UpdatedAt, &a.DeletedAt)

	if err != nil {
		return nil, err
	}

	return &a, nil
}

// DeleteAvatarByID удаляет аватарку по ID (мягкое удаление)
func (r *PostgresRepository) DeleteAvatarByID(ctx context.Context, avatarID string) error {
	_, err := r.db.ExecContext(ctx, "update avatars set deleted_at = $1 where id = $2", time.Now(), avatarID)
	return err
}

// DeleteAvatarByS3Key удаляет аватарку по ключу (мягкое удаление)
func (r *PostgresRepository) DeleteAvatarByS3Key(ctx context.Context, s3Key string) error {
	_, err := r.db.ExecContext(ctx, "update avatars set deleted_at = $1 where s3_key = $2", time.Now(), s3Key)
	return err
}

// DeleteAvatarByUserID удаляет аватарку по ID пользователя (мягкое удаление)
func (r *PostgresRepository) DeleteAvatarByUserID(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, "update avatars set deleted_at = $1 where user_id = $2", time.Now(), userID)
	return err
}

// GetUserAvatars получает список аватарок пользователя
func (r *PostgresRepository) GetUserAvatars(ctx context.Context, userID string) ([]*model.Avatar, error) {
	rows, err := r.db.QueryContext(ctx, `select id, user_id, file_name, mime_type, size_bytes, 
	s3_key, thumbnail_s3_keys, upload_status, processing_status, 
	created_at, updated_at, deleted_at from avatars where user_id = $1`, userID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var avatars []*model.Avatar
	for rows.Next() {
		var a model.Avatar
		err := rows.Scan(&a.ID, &a.UserID, &a.FileName, &a.MimeType, &a.SizeBytes,
			&a.S3Key, &a.ThumbnailS3Keys, &a.UploadStatus, &a.ProcessingStatus,
			&a.CreatedAt, &a.UpdatedAt, &a.DeletedAt)

		if err != nil {
			return nil, err
		}

		avatars = append(avatars, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return avatars, nil
}
