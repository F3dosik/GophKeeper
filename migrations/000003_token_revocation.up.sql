-- Отзыв JWT на сервере.
-- token_version: токен действителен, только пока его версия совпадает с версией пользователя;
-- увеличение версии отзывает все токены пользователя («выйти на всех устройствах»).
ALTER TABLE users ADD COLUMN token_version INTEGER NOT NULL DEFAULT 0;

-- revoked_tokens: отозванные по jti отдельные токены («выйти на этом устройстве»).
-- Записи нужны только до истечения токена и удаляются после expires_at.
CREATE TABLE revoked_tokens (
    jti UUID PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_revoked_tokens_expires_at ON revoked_tokens(expires_at);
