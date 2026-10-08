// Package service содержит бизнес-логику сервера.
package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/internal/server/jwtutil"
	"github.com/F3dosik/GophKeeper/pkg/crypto"
	"github.com/google/uuid"
)

type AuthService interface {
	// Create регистрирует нового пользователя.
	// Возвращает ErrUserAlreadyExists если логин занят.
	Create(ctx context.Context, login string, authKey, salt []byte) error

	// GetSalt возвращает соль пользователя по логину.
	// Для несуществующего логина возвращает детерминированную фиктивную соль,
	// чтобы не раскрывать факт наличия пользователя (защита от перечисления).
	GetSalt(ctx context.Context, login string) ([]byte, error)

	// Login проверяет credentials и возвращает JWT токен.
	// Возвращает ErrInvalidCredentials если authKey неверный или логин не существует
	// (единый код ответа скрывает факт наличия пользователя).
	Login(ctx context.Context, login string, authKey []byte) (string, error)

	// Logout отзывает токен с идентификатором jti (действующий до expiresAt) или,
	// если allSessions == true, все токены пользователя userID.
	Logout(ctx context.Context, userID, jti uuid.UUID, expiresAt time.Time, allSessions bool) error
}

// Контексты HKDF для ключей, выводимых из JWT_SECRET.
const (
	infoTokenSigning = "gophkeeper/jwt-signing"
	infoFakeSalt     = "gophkeeper/fake-salt"
)

// ServerKeys — независимые ключи сервера, выведенные из одного секрета (JWT_SECRET).
// Раздельные ключи гарантируют, что фиктивные соли (их видит любой) ничего не говорят
// о ключе подписи токенов, а смена одного назначения не затрагивает другое.
type ServerKeys struct {
	// TokenSigning — ключ подписи JWT (HMAC-SHA256).
	TokenSigning string
	// FakeSalt — ключ для детерминированных фиктивных солей несуществующих логинов.
	FakeSalt []byte
}

// DeriveServerKeys выводит ServerKeys из секрета через HKDF-SHA256.
func DeriveServerKeys(secret string) (ServerKeys, error) {
	signing, err := crypto.HKDF([]byte(secret), infoTokenSigning)
	if err != nil {
		return ServerKeys{}, fmt.Errorf("derive token signing key: %w", err)
	}
	fakeSalt, err := crypto.HKDF([]byte(secret), infoFakeSalt)
	if err != nil {
		return ServerKeys{}, fmt.Errorf("derive fake salt key: %w", err)
	}
	return ServerKeys{TokenSigning: string(signing), FakeSalt: fakeSalt}, nil
}

// AuthConfig — параметры сервиса аутентификации.
type AuthConfig struct {
	// Keys — ключи подписи токенов и фиктивных солей.
	Keys ServerKeys
	// TokenTTL — время жизни выдаваемых токенов.
	TokenTTL time.Duration
}

// authService реализует AuthService.
type authService struct {
	repo   domain.UserRepository
	tokens domain.TokenRepository
	cfg    AuthConfig
}

// NewAuthService создаёт новый экземпляр authService.
// tokens хранит состояние отзыва токенов, cfg задаёт ключи и время жизни токенов.
func NewAuthService(repo domain.UserRepository, tokens domain.TokenRepository, cfg AuthConfig) AuthService {
	return &authService{repo: repo, tokens: tokens, cfg: cfg}
}

// Create регистрирует нового пользователя.
// В БД сохраняется SHA-256 от authKey, а не сам ключ: утечка БД не позволяет войти под пользователем.
func (s *authService) Create(ctx context.Context, login string, authKey, salt []byte) error {
	return s.repo.Create(ctx, &domain.User{
		Login:        login,
		PasswordHash: crypto.HashAuthKey(authKey),
		PasswordSalt: salt,
	})
}

// GetSalt возвращает соль пользователя по логину.
// Для несуществующего логина возвращает детерминированную соль, выведенную
// из логина и ключа фиктивных солей, чтобы ответ был неотличим от реального пользователя.
func (s *authService) GetSalt(ctx context.Context, login string) ([]byte, error) {
	user, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return crypto.GenerateSaltByLogin(login, s.cfg.Keys.FakeSalt), nil
		}
		return nil, err
	}

	return user.PasswordSalt, nil
}

// Login проверяет credentials и возвращает JWT токен.
// Отсутствие пользователя и неверный authKey возвращают одинаковый ErrInvalidCredentials,
// чтобы скрыть факт существования логина.
func (s *authService) Login(ctx context.Context, login string, authKey []byte) (string, error) {
	user, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", domain.ErrInvalidCredentials
		}
		return "", err
	}

	if subtle.ConstantTimeCompare(user.PasswordHash, crypto.HashAuthKey(authKey)) != 1 {
		return "", domain.ErrInvalidCredentials
	}
	token, err := jwtutil.GenerateToken(user.ID, user.TokenVersion, s.cfg.Keys.TokenSigning, s.cfg.TokenTTL)
	if err != nil {
		return "", fmt.Errorf("authService.Login: generate token: %w", err)
	}

	return token, nil
}

// Logout отзывает текущий токен или все токены пользователя.
func (s *authService) Logout(
	ctx context.Context, userID, jti uuid.UUID, expiresAt time.Time, allSessions bool,
) error {
	if allSessions {
		return s.tokens.RevokeAll(ctx, userID)
	}
	return s.tokens.Revoke(ctx, jti, expiresAt)
}
