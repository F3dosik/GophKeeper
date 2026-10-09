package service

import (
	"bytes"
	"context"
	"github.com/F3dosik/GophKeeper/internal/server/jwtutil"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/internal/server/mocks"
	"github.com/F3dosik/GophKeeper/pkg/crypto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Корректные по длине тестовые ключи и соль.
var (
	testAuthKey  = bytes.Repeat([]byte{1}, authKeyLength)
	testWrongKey = bytes.Repeat([]byte{2}, authKeyLength)
	testSalt     = bytes.Repeat([]byte{3}, saltLength)
)

// testAuthConfig возвращает конфигурацию с ключами, выведенными из тестового секрета.
func testAuthConfig(t *testing.T) AuthConfig {
	t.Helper()
	keys, err := DeriveServerKeys("test-jwt-secret-0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	return AuthConfig{Keys: keys, TokenTTL: time.Hour}
}

func TestDeriveServerKeys(t *testing.T) {
	keys, err := DeriveServerKeys("test-jwt-secret-0123456789abcdef")
	assert.NoError(t, err)
	assert.Len(t, keys.TokenSigning, 32)
	assert.Len(t, keys.FakeSalt, 32)
	assert.NotEqual(t, []byte(keys.TokenSigning), keys.FakeSalt, "keys must be independent")
	assert.NotEqual(t, "test-jwt-secret-0123456789abcdef", keys.TokenSigning, "raw secret must not be used")

	again, err := DeriveServerKeys("test-jwt-secret-0123456789abcdef")
	assert.NoError(t, err)
	assert.Equal(t, keys, again, "derivation must be deterministic across restarts")
}

func TestAuthService_Login_Success(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("GetByLogin", mock.Anything, "user").
		Return(&domain.User{
			ID:           uuid.New(),
			PasswordHash: crypto.HashAuthKey(testAuthKey),
		}, nil)
	svc := NewAuthService(mockRepo, mocks.NewTokenRepository(t), testAuthConfig(t))

	result, err := svc.Login(context.Background(), "user", testAuthKey)

	assert.NoError(t, err)
	assert.NotEmpty(t, result.Token)
	assert.False(t, result.PasswordChangeRequired)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("GetByLogin", mock.Anything, "user").
		Return(&domain.User{
			PasswordHash: crypto.HashAuthKey(testAuthKey),
		}, nil)

	svc := NewAuthService(mockRepo, mocks.NewTokenRepository(t), testAuthConfig(t))

	_, err := svc.Login(context.Background(), "user", testWrongKey)

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("GetByLogin", mock.Anything, "user").
		Return(nil, domain.ErrUserNotFound)

	svc := NewAuthService(mockRepo, mocks.NewTokenRepository(t), testAuthConfig(t))

	_, err := svc.Login(context.Background(), "user", testAuthKey)

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_Create_Success(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("Create", mock.Anything, &domain.User{
		Login:        "user",
		PasswordHash: crypto.HashAuthKey(testAuthKey),
		PasswordSalt: testSalt,
		KDF:          domain.DefaultKDFParams,
	}).Return(nil)

	svc := NewAuthService(mockRepo, mocks.NewTokenRepository(t), testAuthConfig(t))
	_, err := svc.Create(context.Background(), domain.Registration{Login: "user", AuthKey: testAuthKey, Salt: testSalt, KDF: domain.DefaultKDFParams})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Create_AlreadyExists(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("Create", mock.Anything, mock.Anything).
		Return(domain.ErrUserAlreadyExists)

	svc := NewAuthService(mockRepo, mocks.NewTokenRepository(t), testAuthConfig(t))
	_, err := svc.Create(context.Background(), domain.Registration{Login: "user", AuthKey: testAuthKey, Salt: testSalt, KDF: domain.DefaultKDFParams})

	assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
}

func TestAuthService_GetSalt_Success(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("GetByLogin", mock.Anything, "user").
		Return(&domain.User{PasswordSalt: testSalt, KDF: domain.LegacyKDFParams}, nil)

	svc := NewAuthService(mockRepo, mocks.NewTokenRepository(t), testAuthConfig(t))
	salt, kdf, err := svc.GetSalt(context.Background(), "user")

	assert.NoError(t, err)
	assert.Equal(t, testSalt, salt)
	assert.Equal(t, domain.LegacyKDFParams, kdf, "stored params must be returned as is")
}

func TestAuthService_GetSalt_UserNotFound_ReturnsDeterministicFakeSalt(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("GetByLogin", mock.Anything, "user").
		Return(nil, domain.ErrUserNotFound).Times(2)

	svc := NewAuthService(mockRepo, mocks.NewTokenRepository(t), testAuthConfig(t))

	salt1, kdf, err := svc.GetSalt(context.Background(), "user")
	assert.Equal(t, domain.DefaultKDFParams, kdf)
	assert.NoError(t, err)
	assert.Len(t, salt1, 16)

	salt2, _, err := svc.GetSalt(context.Background(), "user")
	assert.NoError(t, err)
	assert.Equal(t, salt1, salt2, "fake salt must be deterministic per login")
}

// Некорректные данные отклоняются до обращения к БД (моки без ожиданий упадут при вызове).
func TestAuthService_RejectsInvalidInput(t *testing.T) {
	svc := NewAuthService(mocks.NewUserRepository(t), mocks.NewTokenRepository(t), testAuthConfig(t))
	ctx := context.Background()

	kdf := domain.DefaultKDFParams
	invalid := map[string]domain.Registration{
		"empty login": {Login: "", AuthKey: testAuthKey, Salt: testSalt, KDF: kdf},
		"no auth key": {Login: "user", Salt: testSalt, KDF: kdf},
		"huge salt":   {Login: "user", AuthKey: testAuthKey, Salt: make([]byte, 1<<20), KDF: kdf},
		"weak kdf":    {Login: "user", AuthKey: testAuthKey, Salt: testSalt, KDF: domain.KDFParams{Time: 1, MemoryKiB: 8, Threads: 1}},
	}
	for name, reg := range invalid {
		_, err := svc.Create(ctx, reg)
		assert.ErrorIs(t, err, domain.ErrInvalidArgument, name)
	}

	_, _, err := svc.GetSalt(ctx, strings.Repeat("a", 1<<20))
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)

	_, err = svc.Login(ctx, "user", []byte("short"))
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

// iterate возвращает SecretIterator по срезу.
func iterate(secrets ...*domain.ReencryptedSecret) domain.SecretIterator {
	i := 0
	return func() (*domain.ReencryptedSecret, error) {
		if i == len(secrets) {
			return nil, nil
		}
		i++
		return secrets[i-1], nil
	}
}

// drain вычитывает итератор до конца, как это делает репозиторий.
func drain(next domain.SecretIterator) error {
	for {
		secret, err := next()
		if err != nil || secret == nil {
			return err
		}
	}
}

func TestAuthService_ChangePassword(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	change := domain.PasswordChange{
		OldAuthKey: testAuthKey, NewAuthKey: testWrongKey, NewSalt: testSalt, NewKDF: domain.DefaultKDFParams,
	}
	valid := &domain.ReencryptedSecret{
		OldBlindIndex: strings.Repeat("a", 64), NewBlindIndex: strings.Repeat("b", 64), Data: []byte("x"),
	}
	cfg := testAuthConfig(t)
	cfg.SecretLimits = SecretLimits{MaxSize: 16, MaxCount: 2}

	t.Run("success stores hashes and issues token with new version", func(t *testing.T) {
		repo := mocks.NewUserRepository(t)
		repo.On("ChangePassword", mock.Anything, userID, domain.PasswordHashChange{
			OldHash: crypto.HashAuthKey(testAuthKey),
			NewHash: crypto.HashAuthKey(testWrongKey),
			NewSalt: testSalt,
			NewKDF:  domain.DefaultKDFParams,
		}, mock.Anything).Return(5, nil).Run(func(args mock.Arguments) {
			require.NoError(t, drain(args.Get(3).(domain.SecretIterator)))
		})

		token, err := NewAuthService(repo, mocks.NewTokenRepository(t), cfg).
			ChangePassword(ctx, userID, change, iterate(valid))
		require.NoError(t, err)

		claims, err := jwtutil.ParseToken(token, cfg.Keys.TokenSigning)
		require.NoError(t, err)
		assert.Equal(t, 5, claims.TokenVersion)
	})

	t.Run("invalid header is rejected before touching the db", func(t *testing.T) {
		svc := NewAuthService(mocks.NewUserRepository(t), mocks.NewTokenRepository(t), cfg)
		bad := change
		bad.NewKDF = domain.KDFParams{Time: 1, MemoryKiB: 8, Threads: 1}
		_, err := svc.ChangePassword(ctx, userID, bad, iterate())
		assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	})

	invalidSecrets := map[string]struct {
		secrets []*domain.ReencryptedSecret
		want    error
	}{
		"bad blind index": {[]*domain.ReencryptedSecret{{OldBlindIndex: "x", NewBlindIndex: valid.NewBlindIndex, Data: []byte("x")}}, domain.ErrInvalidArgument},
		"empty data":      {[]*domain.ReencryptedSecret{{OldBlindIndex: valid.OldBlindIndex, NewBlindIndex: valid.NewBlindIndex}}, domain.ErrInvalidArgument},
		"too large":       {[]*domain.ReencryptedSecret{{OldBlindIndex: valid.OldBlindIndex, NewBlindIndex: valid.NewBlindIndex, Data: make([]byte, 17)}}, domain.ErrSecretTooLarge},
		"too many":        {[]*domain.ReencryptedSecret{valid, valid, valid}, domain.ErrSecretQuotaExceeded},
	}
	for name, tc := range invalidSecrets {
		t.Run(name, func(t *testing.T) {
			repo := mocks.NewUserRepository(t)
			repo.On("ChangePassword", mock.Anything, userID, mock.Anything, mock.Anything).
				Return(0, nil).Run(func(args mock.Arguments) {
				assert.ErrorIs(t, drain(args.Get(3).(domain.SecretIterator)), tc.want)
			})
			_, _ = NewAuthService(repo, mocks.NewTokenRepository(t), cfg).
				ChangePassword(ctx, userID, change, iterate(tc.secrets...))
		})
	}
}

func TestAuthService_Create_Temporary(t *testing.T) {
	cfg := testAuthConfig(t)
	cfg.TempPasswordTTL = 24 * time.Hour

	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
		return u.PasswordExpiresAt != nil && time.Until(*u.PasswordExpiresAt) > 23*time.Hour
	})).Return(nil)

	svc := NewAuthService(mockRepo, mocks.NewTokenRepository(t), cfg)
	expiresAt, err := svc.Create(context.Background(), domain.Registration{
		Login: "user", AuthKey: testAuthKey, Salt: testSalt, KDF: domain.DefaultKDFParams, Temporary: true,
	})
	require.NoError(t, err)
	require.NotNil(t, expiresAt)
}

func TestAuthService_Login_TemporaryPassword(t *testing.T) {
	user := func(expiresAt time.Time) *domain.User {
		return &domain.User{ID: uuid.New(), PasswordHash: crypto.HashAuthKey(testAuthKey), PasswordExpiresAt: &expiresAt}
	}

	t.Run("valid temporary password gives a restricted token", func(t *testing.T) {
		cfg := testAuthConfig(t)
		mockRepo := mocks.NewUserRepository(t)
		mockRepo.On("GetByLogin", mock.Anything, "user").Return(user(time.Now().Add(time.Hour)), nil)

		result, err := NewAuthService(mockRepo, mocks.NewTokenRepository(t), cfg).
			Login(context.Background(), "user", testAuthKey)
		require.NoError(t, err)
		assert.True(t, result.PasswordChangeRequired)

		claims, err := jwtutil.ParseToken(result.Token, cfg.Keys.TokenSigning)
		require.NoError(t, err)
		assert.True(t, claims.PasswordChangeOnly)
		assert.LessOrEqual(t, time.Until(claims.ExpiresAt.Time), passwordChangeTokenTTL)
	})

	t.Run("expired temporary password is rejected", func(t *testing.T) {
		mockRepo := mocks.NewUserRepository(t)
		mockRepo.On("GetByLogin", mock.Anything, "user").Return(user(time.Now().Add(-time.Minute)), nil)

		_, err := NewAuthService(mockRepo, mocks.NewTokenRepository(t), testAuthConfig(t)).
			Login(context.Background(), "user", testAuthKey)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})
}

func TestAuthService_DeleteAccount(t *testing.T) {
	userID := uuid.New()

	t.Run("passes password hash to repository", func(t *testing.T) {
		repo := mocks.NewUserRepository(t)
		repo.On("DeleteWithPassword", mock.Anything, userID, crypto.HashAuthKey(testAuthKey)).Return(nil)

		err := NewAuthService(repo, mocks.NewTokenRepository(t), testAuthConfig(t)).
			DeleteAccount(context.Background(), userID, testAuthKey)
		assert.NoError(t, err)
	})

	t.Run("wrong password", func(t *testing.T) {
		repo := mocks.NewUserRepository(t)
		repo.On("DeleteWithPassword", mock.Anything, userID, mock.Anything).Return(domain.ErrInvalidCredentials)

		err := NewAuthService(repo, mocks.NewTokenRepository(t), testAuthConfig(t)).
			DeleteAccount(context.Background(), userID, testWrongKey)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("malformed key is rejected before the db", func(t *testing.T) {
		err := NewAuthService(mocks.NewUserRepository(t), mocks.NewTokenRepository(t), testAuthConfig(t)).
			DeleteAccount(context.Background(), userID, []byte("short"))
		assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	})
}
