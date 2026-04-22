// Package service реализует бизнес-логику клиента GophKeeper.
// Обеспечивает аутентификацию пользователей, управление секретами,
// а также шифрование и расшифровку данных на стороне клиента.
package service

import (
	"context"
	"fmt"

	"github.com/F3dosik/GophKeeper/internal/client/grpcclient"
	"github.com/F3dosik/GophKeeper/internal/client/session"
	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/pkg/crypto"
)

// AuthService определяет интерфейс для аутентификации пользователя.
type AuthService interface {
	// CreateUser регистрирует нового пользователя на сервере.
	// Генерирует соль, деривирует мастер-ключ через Argon2id и отправляет на сервер.
	CreateUser(ctx context.Context, login, password string) error

	// Login аутентифицирует пользователя и сохраняет сессию (логин + JWT токен) в файл.
	// Запрашивает соль с сервера, деривирует мастер-ключ и получает токен.
	Login(ctx context.Context, login, password string) error

	// DeriveMasterKey запрашивает соль пользователя с сервера и деривирует masterKey
	// через Argon2id. Используется перед операциями с секретами для получения ключа шифрования.
	DeriveMasterKey(ctx context.Context, login, password string) ([]byte, error)
}

// authService реализует AuthService.
type authService struct {
	// client используется для взаимодействия с gRPC сервером аутентификации.
	client grpcclient.AuthClient
	// sessionPath — путь к файлу для хранения сессии пользователя (логин + JWT токен).
	sessionPath string
}

// NewAuthService создаёт новый authService с заданным gRPC клиентом и путём к файлу сессии.
func NewAuthService(client grpcclient.AuthClient, sessionPath string) AuthService {
	return &authService{client: client, sessionPath: sessionPath}
}

// CreateUser регистрирует нового пользователя.
// Генерирует случайную соль, деривирует мастер-ключ через Argon2id и отправляет credentials на сервер.
func (s *authService) CreateUser(ctx context.Context, login, password string) error {
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return fmt.Errorf("authService: %w", err)
	}

	masterKey := crypto.DeriveKey(password, salt)
	if err := s.client.CreateUser(
		ctx, domain.Credentials{Login: login, MasterKey: masterKey}, salt,
	); err != nil {
		return fmt.Errorf("authService.CreateUser: %w", err)
	}

	return nil
}

// Login аутентифицирует пользователя на сервере.
// Запрашивает соль по логину, деривирует мастер-ключ, получает JWT токен
// и сохраняет сессию (логин + токен) в файл для последующих вызовов.
func (s *authService) Login(ctx context.Context, login, password string) error {
	salt, err := s.client.GetSalt(ctx, login)
	if err != nil {
		return fmt.Errorf("authService.Login: %w", err)
	}

	masterKey := crypto.DeriveKey(password, salt)
	token, err := s.client.Login(ctx, domain.Credentials{Login: login, MasterKey: masterKey})
	if err != nil {
		return fmt.Errorf("authService.Login: %w", err)
	}

	if err := session.Save(s.sessionPath, &session.Session{Login: login, Token: token}); err != nil {
		return fmt.Errorf("authService.Login: %w", err)
	}

	return nil
}

// DeriveMasterKey получает соль пользователя с сервера и деривирует masterKey через Argon2id.
// Мастер-ключ далее используется для деривации ключей шифрования секретов на клиенте.
func (s *authService) DeriveMasterKey(ctx context.Context, login, password string) ([]byte, error) {
	salt, err := s.client.GetSalt(ctx, login)
	if err != nil {
		return nil, fmt.Errorf("authService.DeriveMasterKey: %w", err)
	}

	return crypto.DeriveKey(password, salt), nil
}
