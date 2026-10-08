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

	// ChangePassword в одной транзакции заменяет хеш, соль и параметры пароля,
	// перешифровывает все секреты пользователя и увеличивает версию токенов
	// (отзывая все токены). Возвращает новую версию токенов.
	// Возвращает ErrInvalidCredentials, если change.OldHash не совпадает с сохранённым,
	// и ErrSecretsChanged, если набор секретов не совпал с переданным.
	ChangePassword(ctx context.Context, userID uuid.UUID, change PasswordHashChange, secrets SecretIterator) (int, error)

	// DeleteWithPassword удаляет пользователя (и каскадом его секреты), если хеш его
	// пароля равен passwordHash. Возвращает ErrInvalidCredentials, если не совпал
	// или пользователя нет.
	DeleteWithPassword(ctx context.Context, userID uuid.UUID, passwordHash []byte) error

	// DeleteByLogin удаляет пользователя (и каскадом его секреты) по логину.
	// Возвращает ErrUserNotFound, если пользователя нет.
	DeleteByLogin(ctx context.Context, login string) error
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
	// Запись выполняется, только если версия токенов пользователя равна tokenVersion:
	// так секрет не может появиться со старыми ключами после смены пароля.
	// Возвращает ErrSecretAlreadyExists, если секрет с таким blind index уже существует у пользователя,
	// и ErrInvalidCredentials, если версия токенов устарела.
	Create(ctx context.Context, secret *Secret, tokenVersion int) error

	// Update обновляет данные существующего секрета.
	// Секрет идентифицируется по UserID и BlindIndex; как и Create, требует актуальную tokenVersion.
	// Возвращает ErrSecretNotFound, если секрет не найден или версия токенов устарела.
	Update(ctx context.Context, secret *Secret, tokenVersion int) error

	// GetByBlindIndex возвращает секрет по идентификатору пользователя и blind index.
	// Возвращает ErrSecretNotFound, если секрет не найден.
	GetByBlindIndex(ctx context.Context, userID uuid.UUID, blindIndex string) (*Secret, error)

	// ListByUserID возвращает до limit секретов пользователя с ID больше afterID,
	// упорядоченных по ID (uuid.Nil — с начала). Используется для постраничной выдачи.
	// При отсутствии секретов возвращает пустой срез без ошибки.
	ListByUserID(ctx context.Context, userID, afterID uuid.UUID, limit int) ([]*Secret, error)

	// Delete удаляет секрет по идентификатору пользователя и blind index.
	// Возвращает ErrSecretNotFound, если секрет не найден.
	Delete(ctx context.Context, userID uuid.UUID, blindIndex string) error

	// CountByUserID возвращает количество секретов пользователя.
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
}
