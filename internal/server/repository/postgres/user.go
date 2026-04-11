package postgres

import (
	"context"
	"fmt"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/internal/server/repository"
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
		INSERT INTO users (login, password_hash, password_salt)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, user.Login, user.PasswordHash, user.PasswordSalt).Scan(&user.ID, &user.CreatedAt)
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

	err := repository.WithRetry(ctx, isRetriable, func() error {
		return r.pool.QueryRow(ctx, `
			SELECT id, password_hash, password_salt, created_at
			FROM users
			WHERE login = $1
		`, login).Scan(&user.ID, &user.PasswordHash, &user.PasswordSalt, &user.CreatedAt)
	})

	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrUserNotFound
		}

		return nil, fmt.Errorf("userRepository.GetByLogin: %w", err)
	}

	return &user, nil
}
