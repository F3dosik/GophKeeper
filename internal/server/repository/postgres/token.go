package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/internal/server/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// tokenRepository реализует domain.TokenRepository поверх пула соединений PostgreSQL.
type tokenRepository struct {
	pool *pgxpool.Pool
}

// NewTokenRepository создает новый экземпляр репозитория отзыва токенов.
func NewTokenRepository(pool *pgxpool.Pool) domain.TokenRepository {
	return &tokenRepository{pool: pool}
}

// IsActive проверяет версию токена и отсутствие jti среди отозванных одним запросом.
// Для удалённого пользователя возвращает false.
func (r *tokenRepository) IsActive(ctx context.Context, userID uuid.UUID, tokenVersion int, jti uuid.UUID) (bool, error) {
	var active bool
	err := repository.WithRetry(ctx, isRetriable, func() error {
		return r.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM users
				WHERE id = $1 AND token_version = $2
			) AND NOT EXISTS (
				SELECT 1 FROM revoked_tokens WHERE jti = $3
			)
		`, userID, tokenVersion, jti).Scan(&active)
	})
	if err != nil {
		return false, fmt.Errorf("tokenRepository.IsActive: %w", err)
	}
	return active, nil
}

// Revoke добавляет jti в список отозванных и попутно удаляет записи об уже истёкших токенах.
func (r *tokenRepository) Revoke(ctx context.Context, jti uuid.UUID, expiresAt time.Time) error {
	err := repository.WithRetry(ctx, isRetriable, func() error {
		if _, err := r.pool.Exec(ctx, `
			DELETE FROM revoked_tokens WHERE expires_at < now()
		`); err != nil {
			return err
		}
		_, err := r.pool.Exec(ctx, `
			INSERT INTO revoked_tokens (jti, expires_at)
			VALUES ($1, $2)
			ON CONFLICT (jti) DO NOTHING
		`, jti, expiresAt)
		return err
	})
	if err != nil {
		return fmt.Errorf("tokenRepository.Revoke: %w", err)
	}
	return nil
}

// RevokeAll увеличивает версию токенов пользователя, делая недействительными все выданные токены.
func (r *tokenRepository) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	err := repository.WithRetry(ctx, isRetriable, func() error {
		_, err := r.pool.Exec(ctx, `
			UPDATE users SET token_version = token_version + 1 WHERE id = $1
		`, userID)
		return err
	})
	if err != nil {
		return fmt.Errorf("tokenRepository.RevokeAll: %w", err)
	}
	return nil
}
