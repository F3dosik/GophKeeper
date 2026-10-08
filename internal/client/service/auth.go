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
	// Запрашивает соль с сервера, деривирует ключ аутентификации и получает токен.
	Login(ctx context.Context, login, password string) error

	// Unlock проверяет мастер-пароль на сервере и возвращает masterKey для деривации
	// ключей шифрования. Используется перед операциями с секретами: без проверки неверный
	// пароль дал бы другие ключи, и секрет был бы сохранён в недоступном для пользователя виде.
	// Возвращает domain.ErrInvalidCredentials, если пароль неверный.
	Unlock(ctx context.Context, login, password string) ([]byte, error)
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
// Генерирует случайную соль, деривирует мастер-ключ через Argon2id и отправляет на сервер
// производный от него ключ аутентификации (сам masterKey клиент не покидает).
func (s *authService) CreateUser(ctx context.Context, login, password string) error {
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return fmt.Errorf("authService: %w", err)
	}

	authKey, err := deriveAuthKey(password, salt)
	if err != nil {
		return fmt.Errorf("authService.CreateUser: %w", err)
	}
	if err := s.client.CreateUser(
		ctx, domain.Credentials{Login: login, AuthKey: authKey}, salt,
	); err != nil {
		return fmt.Errorf("authService.CreateUser: %w", err)
	}

	return nil
}

// Login аутентифицирует пользователя на сервере.
// Запрашивает соль по логину, деривирует ключ аутентификации, получает JWT токен
// и сохраняет сессию (логин + токен) в файл для последующих вызовов.
func (s *authService) Login(ctx context.Context, login, password string) error {
	salt, err := s.client.GetSalt(ctx, login)
	if err != nil {
		return fmt.Errorf("authService.Login: %w", err)
	}

	authKey, err := deriveAuthKey(password, salt)
	if err != nil {
		return fmt.Errorf("authService.Login: %w", err)
	}
	token, err := s.client.Login(ctx, domain.Credentials{Login: login, AuthKey: authKey})
	if err != nil {
		return fmt.Errorf("authService.Login: %w", err)
	}

	if err := session.Save(s.sessionPath, &session.Session{Login: login, Token: token}); err != nil {
		return fmt.Errorf("authService.Login: %w", err)
	}

	return nil
}

// Unlock получает соль пользователя, деривирует masterKey через Argon2id и проверяет
// пароль, выполняя Login с производным ключом аутентификации. Новый токен не сохраняется:
// цель вызова — только проверка пароля.
func (s *authService) Unlock(ctx context.Context, login, password string) ([]byte, error) {
	salt, err := s.client.GetSalt(ctx, login)
	if err != nil {
		return nil, fmt.Errorf("authService.Unlock: %w", err)
	}

	masterKey := crypto.DeriveKey(password, salt)
	authKey, err := crypto.HKDF(masterKey, crypto.InfoAuth)
	if err != nil {
		return nil, fmt.Errorf("authService.Unlock: %w", err)
	}
	if _, err := s.client.Login(ctx, domain.Credentials{Login: login, AuthKey: authKey}); err != nil {
		return nil, fmt.Errorf("authService.Unlock: %w", err)
	}

	return masterKey, nil
}

// deriveAuthKey вычисляет ключ аутентификации HKDF(Argon2id(password, salt), "auth").
// Ключ отправляется на сервер и независим от ключей шифрования, которые выводятся
// из того же masterKey с другим info.
func deriveAuthKey(password string, salt []byte) ([]byte, error) {
	return crypto.HKDF(crypto.DeriveKey(password, salt), crypto.InfoAuth)
}
