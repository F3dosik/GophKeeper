package postgres

import (
	"context"
	"fmt"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/internal/server/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// secretRepository реализует domain.SecretRepository поверх пула соединений PostgreSQL.
type secretRepository struct {
	pool *pgxpool.Pool
}

// NewSecretRepository создает новый экземпляр репозитория секретов.
func NewSecretRepository(pool *pgxpool.Pool) domain.SecretRepository {
	return &secretRepository{pool: pool}
}

// Create создает новый секрет в базе данных.
// Строка пользователя блокируется FOR SHARE с проверкой версии токенов: если параллельно
// идёт смена пароля, вставка дождётся её завершения и не выполнится, так как версия изменится.
func (r *secretRepository) Create(ctx context.Context, secret *domain.Secret, tokenVersion int) error {
	err := repository.WithRetry(ctx, isRetriable, func() error {
		return r.pool.QueryRow(ctx, `
			WITH u AS (
				SELECT id FROM users WHERE id = $1 AND token_version = $4 FOR SHARE
			)
			INSERT INTO secrets(user_id, blind_index, data)
			SELECT u.id, $2, $3 FROM u
			RETURNING id, updated_at, created_at
		`, secret.UserID, secret.BlindIndex, secret.Data, tokenVersion).Scan(&secret.ID, &secret.UpdatedAt, &secret.CreatedAt)
	})

	if err != nil {
		switch {
		case isNoRows(err):
			return domain.ErrInvalidCredentials
		case isUniqueViolation(err):
			return domain.ErrSecretAlreadyExists
		case isForeignKeyViolation(err):
			return domain.ErrUserNotFound
		}
		return fmt.Errorf("secretRepository.Create: %w", err)
	}

	return nil
}

// Update обновляет существующий секрет в базе данных.
// Как и Create, блокирует строку пользователя и проверяет версию токенов.
func (r *secretRepository) Update(ctx context.Context, secret *domain.Secret, tokenVersion int) error {
	var tag pgconn.CommandTag

	err := repository.WithRetry(ctx, isRetriable, func() error {
		var err error
		tag, err = r.pool.Exec(ctx, `
			WITH u AS (
				SELECT id FROM users WHERE id = $2 AND token_version = $4 FOR SHARE
			)
			UPDATE secrets
			SET data = $1, updated_at = now()
			FROM u
			WHERE secrets.user_id = u.id AND secrets.blind_index = $3
		`, secret.Data, secret.UserID, secret.BlindIndex, tokenVersion)
		return err
	})

	if err != nil {
		return fmt.Errorf("secretRepository.Update: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrSecretNotFound
	}

	return nil
}

// GetByBlindIndex получает секрет по userID и blindIndex.
func (r *secretRepository) GetByBlindIndex(
	ctx context.Context, userID uuid.UUID, blindIndex string,
) (*domain.Secret, error) {
	secret := domain.Secret{UserID: userID, BlindIndex: blindIndex}
	err := repository.WithRetry(ctx, isRetriable, func() error {
		return r.pool.QueryRow(ctx, `
			SELECT id, data, updated_at, created_at
			FROM secrets
			WHERE user_id = $1 AND blind_index = $2
		`, userID, blindIndex).Scan(&secret.ID, &secret.Data, &secret.UpdatedAt, &secret.CreatedAt)
	})

	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrSecretNotFound
		}

		return nil, fmt.Errorf("secretRepository.GetByBlindIndex: %w", err)
	}

	return &secret, nil
}

// ListByUserID возвращает до limit секретов пользователя с ID больше afterID, упорядоченных по ID.
func (r *secretRepository) ListByUserID(ctx context.Context, userID, afterID uuid.UUID, limit int) ([]*domain.Secret, error) {

	var rows pgx.Rows
	err := repository.WithRetry(ctx, isRetriable, func() error {
		var err error
		rows, err = r.pool.Query(ctx, `
			SELECT id, blind_index, data, updated_at, created_at
			FROM secrets
			WHERE user_id = $1 AND id > $2
			ORDER BY id
			LIMIT $3
		`, userID, afterID, limit)

		return err
	})

	if err != nil {
		return nil, fmt.Errorf("secretRepository.ListByUserID: query secrets: %w", err)
	}

	defer rows.Close()

	secrets := make([]*domain.Secret, 0)
	for rows.Next() {
		secret := domain.Secret{UserID: userID}
		err := rows.Scan(
			&secret.ID, &secret.BlindIndex, &secret.Data,
			&secret.UpdatedAt, &secret.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("secretRepository.ListByUserID: scan secret: %w", err)
		}
		secrets = append(secrets, &secret)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("secretRepository.ListByUserID: rows err: %w", err)
	}
	return secrets, nil
}

// Delete удаляет секрет по userID и blindIndex.
func (r *secretRepository) Delete(ctx context.Context, userID uuid.UUID, blindIndex string) error {
	var tag pgconn.CommandTag
	err := repository.WithRetry(ctx, isRetriable, func() error {
		var err error
		tag, err = r.pool.Exec(ctx, `
			DELETE FROM secrets
			WHERE user_id = $1 AND blind_index = $2
		`, userID, blindIndex)
		return err
	})

	if err != nil {
		return fmt.Errorf("secretRepository.Delete: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrSecretNotFound
	}

	return nil
}

// CountByUserID возвращает количество секретов пользователя.
func (r *secretRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := repository.WithRetry(ctx, isRetriable, func() error {
		return r.pool.QueryRow(ctx, `
			SELECT count(*) FROM secrets WHERE user_id = $1
		`, userID).Scan(&count)
	})
	if err != nil {
		return 0, fmt.Errorf("secretRepository.CountByUserID: %w", err)
	}
	return count, nil
}
