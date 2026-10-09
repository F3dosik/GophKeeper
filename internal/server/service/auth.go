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
	// Для временного пароля (reg.Temporary) возвращает время, до которого им можно войти.
	Create(ctx context.Context, reg domain.Registration) (*time.Time, error)

	// GetSalt возвращает соль пользователя по логину.
	// Для несуществующего логина возвращает детерминированную фиктивную соль,
	// чтобы не раскрывать факт наличия пользователя (защита от перечисления).
	// Возвращает соль и параметры Argon2id пользователя.
	GetSalt(ctx context.Context, login string) ([]byte, domain.KDFParams, error)

	// Login проверяет credentials и возвращает JWT токен.
	// Возвращает ErrInvalidCredentials если authKey неверный или логин не существует
	// (единый код ответа скрывает факт наличия пользователя).
	// Для временного пароля возвращает токен, разрешающий только смену пароля, и
	// PasswordChangeRequired; истёкший временный пароль даёт ErrInvalidCredentials.
	Login(ctx context.Context, login string, authKey []byte) (domain.LoginResult, error)

	// Logout отзывает токен с идентификатором jti (действующий до expiresAt) или,
	// если allSessions == true, все токены пользователя userID.
	Logout(ctx context.Context, userID, jti uuid.UUID, expiresAt time.Time, allSessions bool) error

	// ChangePassword проверяет текущий пароль по change.OldAuthKey, сохраняет новые
	// соль, параметры и хеш и заменяет все секреты пользователя перешифрованными из secrets.
	// Всё выполняется атомарно; все прежние токены отзываются. Возвращает новый токен.
	// Возвращает ErrInvalidCredentials при неверном текущем пароле и ErrSecretsChanged,
	// если секреты изменились во время смены.
	ChangePassword(ctx context.Context, userID uuid.UUID, change domain.PasswordChange, secrets domain.SecretIterator) (string, error)

	// DeleteAccount удаляет учётку userID со всеми секретами, если authKey
	// соответствует текущему паролю. Возвращает ErrInvalidCredentials иначе.
	DeleteAccount(ctx context.Context, userID uuid.UUID, authKey []byte) error
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
	// SecretLimits — квоты на секреты; проверяются и при перешифровке во время смены пароля.
	SecretLimits SecretLimits
	// TempPasswordTTL — срок действия временного пароля.
	TempPasswordTTL time.Duration
}

// passwordChangeTokenTTL — срок жизни токена, выдаваемого по временному паролю:
// его хватает только на то, чтобы ввести новый пароль.
const passwordChangeTokenTTL = 10 * time.Minute

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
func (s *authService) Create(ctx context.Context, reg domain.Registration) (*time.Time, error) {
	if err := validateLogin(reg.Login); err != nil {
		return nil, err
	}
	if err := validateAuthKey(reg.AuthKey); err != nil {
		return nil, err
	}
	if err := validateSalt(reg.Salt); err != nil {
		return nil, err
	}
	if err := reg.KDF.Validate(); err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if reg.Temporary {
		t := time.Now().Add(s.cfg.TempPasswordTTL).UTC()
		expiresAt = &t
	}

	err := s.repo.Create(ctx, &domain.User{
		Login:             reg.Login,
		PasswordHash:      crypto.HashAuthKey(reg.AuthKey),
		PasswordSalt:      reg.Salt,
		KDF:               reg.KDF,
		PasswordExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, err
	}
	return expiresAt, nil
}

// GetSalt возвращает соль пользователя по логину.
// Для несуществующего логина возвращает детерминированную соль, выведенную
// из логина и ключа фиктивных солей, чтобы ответ был неотличим от реального пользователя.
func (s *authService) GetSalt(ctx context.Context, login string) ([]byte, domain.KDFParams, error) {
	if err := validateLogin(login); err != nil {
		return nil, domain.KDFParams{}, err
	}
	user, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Фиктивный пользователь получает параметры, с которыми сейчас создаются новые.
			return crypto.GenerateSaltByLogin(login, s.cfg.Keys.FakeSalt), domain.DefaultKDFParams, nil
		}
		return nil, domain.KDFParams{}, err
	}

	return user.PasswordSalt, user.KDF, nil
}

// Login проверяет credentials и возвращает JWT токен.
// Отсутствие пользователя и неверный authKey возвращают одинаковый ErrInvalidCredentials,
// чтобы скрыть факт существования логина.
func (s *authService) Login(ctx context.Context, login string, authKey []byte) (domain.LoginResult, error) {
	if err := validateLogin(login); err != nil {
		return domain.LoginResult{}, err
	}
	if err := validateAuthKey(authKey); err != nil {
		return domain.LoginResult{}, err
	}
	user, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.LoginResult{}, domain.ErrInvalidCredentials
		}
		return domain.LoginResult{}, err
	}

	if subtle.ConstantTimeCompare(user.PasswordHash, crypto.HashAuthKey(authKey)) != 1 {
		return domain.LoginResult{}, domain.ErrInvalidCredentials
	}

	if user.PasswordExpiresAt != nil {
		// Истёкший временный пароль неотличим от неверного: администратору нужно
		// пересоздать учётку.
		if time.Now().After(*user.PasswordExpiresAt) {
			return domain.LoginResult{}, domain.ErrInvalidCredentials
		}
		token, err := jwtutil.GeneratePasswordChangeToken(
			user.ID, user.TokenVersion, s.cfg.Keys.TokenSigning, min(passwordChangeTokenTTL, s.cfg.TokenTTL),
		)
		if err != nil {
			return domain.LoginResult{}, fmt.Errorf("authService.Login: generate token: %w", err)
		}
		return domain.LoginResult{Token: token, PasswordChangeRequired: true}, nil
	}

	token, err := jwtutil.GenerateToken(user.ID, user.TokenVersion, s.cfg.Keys.TokenSigning, s.cfg.TokenTTL)
	if err != nil {
		return domain.LoginResult{}, fmt.Errorf("authService.Login: generate token: %w", err)
	}

	return domain.LoginResult{Token: token}, nil
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

// ChangePassword меняет пароль пользователя и перешифровывает его секреты.
func (s *authService) ChangePassword(
	ctx context.Context, userID uuid.UUID, change domain.PasswordChange, secrets domain.SecretIterator,
) (string, error) {
	if err := validateAuthKey(change.OldAuthKey); err != nil {
		return "", err
	}
	if err := validateAuthKey(change.NewAuthKey); err != nil {
		return "", err
	}
	if err := validateSalt(change.NewSalt); err != nil {
		return "", err
	}
	if err := change.NewKDF.Validate(); err != nil {
		return "", err
	}

	version, err := s.repo.ChangePassword(ctx, userID, domain.PasswordHashChange{
		OldHash: crypto.HashAuthKey(change.OldAuthKey),
		NewHash: crypto.HashAuthKey(change.NewAuthKey),
		NewSalt: change.NewSalt,
		NewKDF:  change.NewKDF,
	}, s.validatedSecrets(secrets))
	if err != nil {
		return "", err
	}

	token, err := jwtutil.GenerateToken(userID, version, s.cfg.Keys.TokenSigning, s.cfg.TokenTTL)
	if err != nil {
		return "", fmt.Errorf("authService.ChangePassword: generate token: %w", err)
	}
	return token, nil
}

// validatedSecrets проверяет каждый перешифрованный секрет по мере получения:
// формат blind index, размер данных и общее число секретов.
func (s *authService) validatedSecrets(next domain.SecretIterator) domain.SecretIterator {
	count := 0
	return func() (*domain.ReencryptedSecret, error) {
		secret, err := next()
		if err != nil || secret == nil {
			return secret, err
		}
		count++
		if count > s.cfg.SecretLimits.MaxCount {
			return nil, domain.ErrSecretQuotaExceeded
		}
		if err := validateBlindIndex(secret.OldBlindIndex); err != nil {
			return nil, err
		}
		if err := validateBlindIndex(secret.NewBlindIndex); err != nil {
			return nil, err
		}
		if len(secret.Data) == 0 {
			return nil, fmt.Errorf("%w: secret data is empty", domain.ErrInvalidArgument)
		}
		if len(secret.Data) > s.cfg.SecretLimits.MaxSize {
			return nil, domain.ErrSecretTooLarge
		}
		return secret, nil
	}
}

// DeleteAccount удаляет учётку пользователя после проверки пароля.
func (s *authService) DeleteAccount(ctx context.Context, userID uuid.UUID, authKey []byte) error {
	if err := validateAuthKey(authKey); err != nil {
		return err
	}
	return s.repo.DeleteWithPassword(ctx, userID, crypto.HashAuthKey(authKey))
}
