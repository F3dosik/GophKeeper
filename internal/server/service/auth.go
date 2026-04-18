// Package service содержит бизнес-логику сервера.
package service

import (
	"context"
	"crypto/subtle"
	"fmt"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/internal/server/jwtutil"
)

type AuthService interface {
	// Create регистрирует нового пользователя.
	// Возвращает ErrUserAlreadyExists если логин занят.
	Create(ctx context.Context, login string, masterKey, salt []byte) error

	// GetSalt возвращает соль пользователя по логину.
	// Возвращает ErrUserNotFound если пользователь не найден.
	GetSalt(ctx context.Context, login string) ([]byte, error)

	// Login проверяет credentials и возвращает JWT токен.
	// Возвращает ErrInvalidCredentials если masterKey неверный.
	Login(ctx context.Context, login string, masterKey []byte) (string, error)
}

// authService реализует AuthService.
type authService struct {
	repo      domain.UserRepository
	jwtSecret string
}

// NewAuthService создаёт новый экземпляр authService.
// jwtSecret используется для подписи JWT токенов.
func NewAuthService(repo domain.UserRepository, jwtSecret string) AuthService {
	return &authService{repo: repo, jwtSecret: jwtSecret}
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
func (s *authService) GetSalt(ctx context.Context, login string) ([]byte, error) {
	user, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		return nil, err
	}

	return user.PasswordSalt, nil
}

// Login проверяет credentials и возвращает JWT токен.
func (s *authService) Login(ctx context.Context, login string, masterKey []byte) (string, error) {
	user, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		return "", err
	}

	if subtle.ConstantTimeCompare(user.PasswordHash, masterKey) != 1 {
		return "", domain.ErrInvalidCredentials
	}
	token, err := jwtutil.GenerateToken(user.ID, s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("authService.Login: generate token: %w", err)
	}

	return token, nil
}
