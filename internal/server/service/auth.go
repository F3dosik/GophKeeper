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
)

type AuthService interface {
	// Create регистрирует нового пользователя.
	// Возвращает ErrUserAlreadyExists если логин занят.
	Create(ctx context.Context, login string, masterKey, salt []byte) error

	// GetSalt возвращает соль пользователя по логину.
	// Для несуществующего логина возвращает детерминированную фиктивную соль,
	// чтобы не раскрывать факт наличия пользователя (защита от перечисления).
	GetSalt(ctx context.Context, login string) ([]byte, error)

	// Login проверяет credentials и возвращает JWT токен.
	// Возвращает ErrInvalidCredentials если masterKey неверный или логин не существует
	// (единый код ответа скрывает факт наличия пользователя).
	Login(ctx context.Context, login string, masterKey []byte) (string, error)
}

// authService реализует AuthService.
type authService struct {
	repo      domain.UserRepository
	jwtSecret string
	tokenTTL  time.Duration
}

// NewAuthService создаёт новый экземпляр authService.
// jwtSecret используется для подписи JWT токенов, tokenTTL задаёт время жизни токена.
func NewAuthService(repo domain.UserRepository, jwtSecret string, tokenTTL time.Duration) AuthService {
	return &authService{repo: repo, jwtSecret: jwtSecret, tokenTTL: tokenTTL}
}

// Create регистрирует нового пользователя.
func (s *authService) Create(ctx context.Context, login string, masterKey, salt []byte) error {
	return s.repo.Create(ctx, &domain.User{
		Login:        login,
		PasswordHash: masterKey,
		PasswordSalt: salt,
	})
}

// GetSalt возвращает соль пользователя по логину.
// Для несуществующего логина возвращает детерминированную соль, выведенную
// из логина и jwtSecret, чтобы ответ был неотличим от реального пользователя.
func (s *authService) GetSalt(ctx context.Context, login string) ([]byte, error) {
	user, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return crypto.GenerateSaltByLogin(login, []byte(s.jwtSecret)), nil
		}
		return nil, err
	}

	return user.PasswordSalt, nil
}

// Login проверяет credentials и возвращает JWT токен.
// Отсутствие пользователя и неверный masterKey возвращают одинаковый ErrInvalidCredentials,
// чтобы скрыть факт существования логина.
func (s *authService) Login(ctx context.Context, login string, masterKey []byte) (string, error) {
	user, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", domain.ErrInvalidCredentials
		}
		return "", err
	}

	if subtle.ConstantTimeCompare(user.PasswordHash, masterKey) != 1 {
		return "", domain.ErrInvalidCredentials
	}
	token, err := jwtutil.GenerateToken(user.ID, s.jwtSecret, s.tokenTTL)
	if err != nil {
		return "", fmt.Errorf("authService.Login: generate token: %w", err)
	}

	return token, nil
}
