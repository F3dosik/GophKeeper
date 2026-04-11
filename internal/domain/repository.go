package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository определяет методы для работы с пользователями в хранилище.
type UserRepository interface {
	// Create сохраняет нового пользователя и заполняет поля ID и CreatedAt.
	// Возвращает ErrUserAlreadyExists, если пользователь с таким логином уже существует.
	Create(ctx context.Context, user *User) error

	// GetByLogin возвращает пользователя по логину.
	// Возвращает ErrUserNotFound, если пользователь не найден.
	GetByLogin(ctx context.Context, login string) (*User, error)
}

// SecretRepository определяет методы для работы с зашифрованными секретами в хранилище.
type SecretRepository interface {
	// Create сохраняет новый секрет и заполняет поля ID, UpdatedAt и CreatedAt.
	// Возвращает ErrSecretAlreadyExists, если секрет с таким blind index уже существует у пользователя.
	Create(ctx context.Context, secret *Secret) error

	// Update обновляет данные существующего секрета.
	// Секрет идентифицируется по UserID и BlindIndex.
	// Возвращает ErrSecretNotFound, если секрет не найден.
	Update(ctx context.Context, secret *Secret) error

	// GetByBlindIndex возвращает секрет по идентификатору пользователя и blind index.
	// Возвращает ErrSecretNotFound, если секрет не найден.
	GetByBlindIndex(ctx context.Context, userID uuid.UUID, blindIndex string) (*Secret, error)

	// ListByUserID возвращает все секреты, принадлежащие пользователю с указанным ID.
	// При отсутствии секретов возвращает пустой срез без ошибки.
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*Secret, error)

	// Delete удаляет секрет по идентификатору пользователя и blind index.
	// Возвращает ErrSecretNotFound, если секрет не найден.
	Delete(ctx context.Context, userID uuid.UUID, blindIndex string) error
}
