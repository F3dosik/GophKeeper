// Package service реализует бизнес-логику клиента GophKeeper.
// Обеспечивает аутентификацию пользователей, управление секретами,
// а также шифрование и расшифровку данных на стороне клиента.
package service

import (
	"context"
	"errors"
	"fmt"
	"os"

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

	// Logout отзывает токен на сервере и удаляет локальную сессию.
	// allSessions == true отзывает все токены пользователя (выход на всех устройствах).
	Logout(ctx context.Context, allSessions bool) error
}

// authService реализует AuthService.
type authService struct {
	// client используется для взаимодействия с gRPC сервером аутентификации.
	client grpcclient.AuthClient
	// sessionPath — путь к файлу для хранения сессии пользователя (логин + JWT токен).
	sessionPath string
	// tokens — токен, с которым соединение выполняет запросы; обновляется после входа.
	tokens *grpcclient.TokenStore
}

// NewAuthService создаёт новый authService с заданным gRPC клиентом, путём к файлу сессии
// и хранилищем токена соединения (может быть nil, если обновлять токен в соединении не нужно).
func NewAuthService(client grpcclient.AuthClient, sessionPath string, tokens *grpcclient.TokenStore) AuthService {
	return &authService{client: client, sessionPath: sessionPath, tokens: tokens}
}

// CreateUser регистрирует нового пользователя.
// Генерирует случайную соль, деривирует мастер-ключ через Argon2id и отправляет на сервер
// производный от него ключ аутентификации (сам masterKey клиент не покидает).
func (s *authService) CreateUser(ctx context.Context, login, password string) error {
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return fmt.Errorf("authService: %w", err)
	}

	kdf := domain.DefaultKDFParams
	_, authKey, err := deriveKeys(password, salt, kdf)
	if err != nil {
		return fmt.Errorf("authService.CreateUser: %w", err)
	}
	if err := s.client.CreateUser(
		ctx, domain.Credentials{Login: login, AuthKey: authKey}, salt, kdf,
	); err != nil {
		return fmt.Errorf("authService.CreateUser: %w", err)
	}

	return nil
}

// Login аутентифицирует пользователя на сервере.
// Запрашивает соль по логину, деривирует ключ аутентификации, получает JWT токен
// и сохраняет сессию (логин + токен) в файл для последующих вызовов.
func (s *authService) Login(ctx context.Context, login, password string) error {
	_, authKey, err := s.userKeys(ctx, login, password)
	if err != nil {
		return fmt.Errorf("authService.Login: %w", err)
	}
	token, err := s.client.Login(ctx, domain.Credentials{Login: login, AuthKey: authKey})
	if err != nil {
		return fmt.Errorf("authService.Login: %w", err)
	}

	if err := s.saveSession(login, token); err != nil {
		return fmt.Errorf("authService.Login: %w", err)
	}

	return nil
}

// Unlock получает соль пользователя, деривирует masterKey через Argon2id и проверяет
// пароль, выполняя Login с производным ключом аутентификации. Полученный новый токен
// сохраняется в сессию и используется соединением, поэтому короткий срок жизни токена
// не требует частого повторного входа.
func (s *authService) Unlock(ctx context.Context, login, password string) ([]byte, error) {
	masterKey, authKey, err := s.userKeys(ctx, login, password)
	if err != nil {
		return nil, fmt.Errorf("authService.Unlock: %w", err)
	}
	token, err := s.client.Login(ctx, domain.Credentials{Login: login, AuthKey: authKey})
	if err != nil {
		return nil, fmt.Errorf("authService.Unlock: %w", err)
	}
	if err := s.saveSession(login, token); err != nil {
		return nil, fmt.Errorf("authService.Unlock: %w", err)
	}

	return masterKey, nil
}

// userKeys запрашивает соль и параметры Argon2id пользователя и выводит из пароля
// мастер-ключ и ключ аутентификации. Параметры от сервера проверяются: подменённый
// сервер мог бы прислать слабые параметры, чтобы удешевить перебор пароля по authKey,
// или огромные, чтобы клиент завис.
func (s *authService) userKeys(ctx context.Context, login, password string) (masterKey, authKey []byte, err error) {
	salt, kdf, err := s.client.GetSalt(ctx, login)
	if err != nil {
		return nil, nil, err
	}
	if err := kdf.Validate(); err != nil {
		return nil, nil, fmt.Errorf("server sent unacceptable kdf params: %w", err)
	}
	return deriveKeys(password, salt, kdf)
}

// deriveKeys выводит мастер-ключ Argon2id(password, salt) и ключ аутентификации
// HKDF(masterKey, "auth"). Ключ аутентификации отправляется на сервер и независим от
// ключей шифрования, которые выводятся из того же masterKey с другим info.
func deriveKeys(password string, salt []byte, kdf domain.KDFParams) (masterKey, authKey []byte, err error) {
	masterKey = crypto.DeriveKey(password, salt, kdf)
	authKey, err = crypto.HKDF(masterKey, crypto.InfoAuth)
	if err != nil {
		return nil, nil, err
	}
	return masterKey, authKey, nil
}

// Logout отзывает токен на сервере, затем удаляет файл сессии.
// Если отзывается только текущий токен, а сервер его уже не принимает (истёк или отозван),
// локальная сессия всё равно удаляется: такой токен и так бесполезен.
func (s *authService) Logout(ctx context.Context, allSessions bool) error {
	err := s.client.Logout(ctx, allSessions)
	if err != nil && (allSessions || !errors.Is(err, domain.ErrInvalidCredentials)) {
		return fmt.Errorf("authService.Logout: %w", err)
	}

	if err := os.Remove(s.sessionPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("authService.Logout: %w", err)
	}
	s.tokens.SetToken("")
	return nil
}

// saveSession сохраняет сессию в файл и передаёт токен соединению.
func (s *authService) saveSession(login, token string) error {
	if err := session.Save(s.sessionPath, &session.Session{Login: login, Token: token}); err != nil {
		return err
	}
	s.tokens.SetToken(token)
	return nil
}
