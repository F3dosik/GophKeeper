package service

import (
	"bytes"
	"context"
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

	token, err := svc.Login(context.Background(), "user", testAuthKey)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
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
	}).Return(nil)

	svc := NewAuthService(mockRepo, mocks.NewTokenRepository(t), testAuthConfig(t))
	err := svc.Create(context.Background(), "user", testAuthKey, testSalt)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Create_AlreadyExists(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("Create", mock.Anything, mock.Anything).
		Return(domain.ErrUserAlreadyExists)

	svc := NewAuthService(mockRepo, mocks.NewTokenRepository(t), testAuthConfig(t))
	err := svc.Create(context.Background(), "user", testAuthKey, testSalt)

	assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
}

func TestAuthService_GetSalt_Success(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("GetByLogin", mock.Anything, "user").
		Return(&domain.User{PasswordSalt: testSalt}, nil)

	svc := NewAuthService(mockRepo, mocks.NewTokenRepository(t), testAuthConfig(t))
	salt, err := svc.GetSalt(context.Background(), "user")

	assert.NoError(t, err)
	assert.Equal(t, testSalt, salt)
}

func TestAuthService_GetSalt_UserNotFound_ReturnsDeterministicFakeSalt(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("GetByLogin", mock.Anything, "user").
		Return(nil, domain.ErrUserNotFound).Times(2)

	svc := NewAuthService(mockRepo, mocks.NewTokenRepository(t), testAuthConfig(t))

	salt1, err := svc.GetSalt(context.Background(), "user")
	assert.NoError(t, err)
	assert.Len(t, salt1, 16)

	salt2, err := svc.GetSalt(context.Background(), "user")
	assert.NoError(t, err)
	assert.Equal(t, salt1, salt2, "fake salt must be deterministic per login")
}

// Некорректные данные отклоняются до обращения к БД (моки без ожиданий упадут при вызове).
func TestAuthService_RejectsInvalidInput(t *testing.T) {
	svc := NewAuthService(mocks.NewUserRepository(t), mocks.NewTokenRepository(t), testAuthConfig(t))
	ctx := context.Background()

	assert.ErrorIs(t, svc.Create(ctx, "", testAuthKey, testSalt), domain.ErrInvalidArgument)
	assert.ErrorIs(t, svc.Create(ctx, "user", nil, testSalt), domain.ErrInvalidArgument)
	assert.ErrorIs(t, svc.Create(ctx, "user", testAuthKey, make([]byte, 1<<20)), domain.ErrInvalidArgument)

	_, err := svc.GetSalt(ctx, strings.Repeat("a", 1<<20))
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)

	_, err = svc.Login(ctx, "user", []byte("short"))
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}
