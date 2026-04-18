// Package service реализует бизнес-логику клиента GophKeeper.
// Обеспечивает аутентификацию пользователей, управление секретами,
// а также шифрование и расшифровку данных на стороне клиента.
package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/F3dosik/GophKeeper/internal/client/grpcclient"
	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/pkg/crypto"
)

// AuthService определяет интерфейс для аутентификации пользователя.
type AuthService interface {
	// CreateUser регистрирует нового пользователя на сервере.
	// Генерирует соль, деривирует мастер-ключ через Argon2id и отправляет на сервер.
	CreateUser(ctx context.Context, login, password string) error

	// Login аутентифицирует пользователя и сохраняет JWT токен в файл.
	// Запрашивает соль с сервера, деривирует мастер-ключ и получает токен.
	Login(ctx context.Context, login, password string) error
}

// authService реализует AuthService
type authService struct {
	// client используется для взаимодействия с gRPC сервером аутентификации.
	client grpcclient.AuthClient
	// tokenPath — путь к файлу для хранения JWT токена.
	tokenPath string
}

// NewAuthService создаёт новый authService с заданным gRPC клиентом и путём к файлу токена.
func NewAuthService(client grpcclient.AuthClient, tokenPath string) AuthService {
	return &authService{client: client, tokenPath: tokenPath}
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
// Запрашивает соль по логину, деривирует мастер-ключ и сохраняет полученный JWT токен в файл.
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

	if err := s.saveToken(token); err != nil {
		return fmt.Errorf("authService.Login: %w", err)
	}

	return nil
}

// saveToken сохраняет JWT токен в файл по пути tokenPath.
// Создаёт директорию если она не существует.
func (s *authService) saveToken(token string) error {
	if err := os.MkdirAll(filepath.Dir(s.tokenPath), 0700); err != nil {
		return fmt.Errorf("save token: create dir: %w", err)
	}
	return os.WriteFile(s.tokenPath, []byte(token), 0600)
}
