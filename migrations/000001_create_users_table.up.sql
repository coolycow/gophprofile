-- Создаём расширение pgcrypto для генерации UUID
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Таблица для хранения пользователей
-- Поддерживается мягкое удаление записей
-- В репозитории поддерживается полное удаление записей
CREATE TABLE users (
    -- id пользователя
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- email пользователя
    email VARCHAR(255) NOT NULL UNIQUE,
    -- пароль пользователя
    password BYTEA NOT NULL,

    -- временные метки создания, обновления и удаления записи
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE NULL DEFAULT NULL
);

CREATE UNIQUE INDEX idx_users_email_unique ON users(email);