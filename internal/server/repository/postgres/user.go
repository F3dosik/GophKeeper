package postgres

import (
	"context"
	"fmt"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/internal/server/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// userRepository реализует domain.UserRepository поверх пула соединений PostgreSQL.
type userRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository создает новый экземпляр репозитория пользователей.
func NewUserRepository(pool *pgxpool.Pool) domain.UserRepository {
	return &userRepository{pool: pool}
}

// Create создает нового пользователя в базе данных.
func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	err := repository.WithRetry(ctx, isRetriable, func() error {
		return r.pool.QueryRow(ctx, `
		INSERT INTO users (login, password_hash, password_salt, kdf_time, kdf_memory_kib, kdf_threads,
		                   password_expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, user.Login, user.PasswordHash, user.PasswordSalt,
			int64(user.KDF.Time), int64(user.KDF.MemoryKiB), int16(user.KDF.Threads), user.PasswordExpiresAt,
		).Scan(&user.ID, &user.CreatedAt)
	})

	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrUserAlreadyExists
		}

		return fmt.Errorf("userRepository.Create: %w", err)
	}

	return nil
}

// GetByLogin получает пользователя по логину.
func (r *userRepository) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	user := domain.User{Login: login}
	var kdfTime, kdfMemory int64
	var kdfThreads int16

	err := repository.WithRetry(ctx, isRetriable, func() error {
		return r.pool.QueryRow(ctx, `
			SELECT id, password_hash, password_salt, kdf_time, kdf_memory_kib, kdf_threads,
			       token_version, password_expires_at, created_at
			FROM users
			WHERE login = $1
		`, login).Scan(&user.ID, &user.PasswordHash, &user.PasswordSalt, &kdfTime, &kdfMemory, &kdfThreads,
			&user.TokenVersion, &user.PasswordExpiresAt, &user.CreatedAt)
	})
	user.KDF = domain.KDFParams{Time: uint32(kdfTime), MemoryKiB: uint32(kdfMemory), Threads: uint8(kdfThreads)}

	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrUserNotFound
		}

		return nil, fmt.Errorf("userRepository.GetByLogin: %w", err)
	}

	return &user, nil
}

// ChangePassword меняет пароль и перешифровывает секреты в одной транзакции.
//
// Сначала обновляется строка пользователя (условие на старый хеш): она остаётся
// заблокированной до конца транзакции, поэтому параллельные Create/Update секретов
// ждут и затем отклоняются из-за новой версии токенов. Каждый секрет обновляется
// по старому blind index и ожидаемому updated_at; в конце число секретов должно
// совпасть с числом переданных — иначе набор изменился и транзакция откатывается.
func (r *userRepository) ChangePassword(
	ctx context.Context, userID uuid.UUID, change domain.PasswordHashChange, secrets domain.SecretIterator,
) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("userRepository.ChangePassword: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var version int
	err = tx.QueryRow(ctx, `
		UPDATE users
		SET password_hash = $3, password_salt = $4,
		    kdf_time = $5, kdf_memory_kib = $6, kdf_threads = $7,
		    token_version = token_version + 1, password_expires_at = NULL
		WHERE id = $1 AND password_hash = $2
		RETURNING token_version
	`, userID, change.OldHash, change.NewHash, change.NewSalt,
		int64(change.NewKDF.Time), int64(change.NewKDF.MemoryKiB), int16(change.NewKDF.Threads),
	).Scan(&version)
	if err != nil {
		if isNoRows(err) {
			return 0, domain.ErrInvalidCredentials
		}
		return 0, fmt.Errorf("userRepository.ChangePassword: update user: %w", err)
	}

	count := 0
	for {
		secret, err := secrets()
		if err != nil {
			return 0, err
		}
		if secret == nil {
			break
		}
		count++

		tag, err := tx.Exec(ctx, `
			UPDATE secrets
			SET blind_index = $3, data = $4, updated_at = now()
			WHERE user_id = $1 AND blind_index = $2 AND updated_at = $5
		`, userID, secret.OldBlindIndex, secret.NewBlindIndex, secret.Data, secret.ExpectedUpdatedAt)
		if err != nil {
			if isUniqueViolation(err) {
				return 0, domain.ErrSecretsChanged
			}
			return 0, fmt.Errorf("userRepository.ChangePassword: update secret: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return 0, domain.ErrSecretsChanged
		}
	}

	var total int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM secrets WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return 0, fmt.Errorf("userRepository.ChangePassword: count secrets: %w", err)
	}
	if total != count {
		return 0, domain.ErrSecretsChanged
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("userRepository.ChangePassword: commit: %w", err)
	}
	return version, nil
}
