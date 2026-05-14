-- Refresh-токены: непрозрачная строка у клиента, в БД — SHA-256 от этой строки.
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- первичный ключ
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- идентификатор пользователя
    token_hash BYTEA NOT NULL, -- хэш непрозрачной строки
    expires_at TIMESTAMPTZ NOT NULL, -- срок действия токена
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW() -- временная метка создания токена
);

CREATE UNIQUE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
