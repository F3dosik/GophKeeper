package domain

import (
	"context"
	"time"

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

// TokenRepository хранит состояние отзыва JWT-токенов.
type TokenRepository interface {
	// IsActive сообщает, действителен ли токен: пользователь существует, версия токена
	// совпадает с текущей версией пользователя и jti не отозван.
	IsActive(ctx context.Context, userID uuid.UUID, tokenVersion int, jti uuid.UUID) (bool, error)

	// Revoke отзывает один токен по jti. expiresAt — время истечения токена,
	// после которого запись об отзыве можно удалить.
	Revoke(ctx context.Context, jti uuid.UUID, expiresAt time.Time) error

	// RevokeAll отзывает все токены пользователя, увеличивая его версию токенов.
	RevokeAll(ctx context.Context, userID uuid.UUID) error
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

	// CountByUserID возвращает количество секретов пользователя.
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
}
