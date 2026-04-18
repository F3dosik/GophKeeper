package service_test

import (
	"context"
	"os"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/client/mocks"
	"github.com/F3dosik/GophKeeper/internal/client/service"
	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAuthService_CreateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("CreateUser", mock.Anything, mock.MatchedBy(func(creds domain.Credentials) bool {
			return creds.Login == "user" && len(creds.MasterKey) == 32
		}), mock.AnythingOfType("[]uint8")).Return(nil)

		svc := service.NewAuthService(mockAuth, t.TempDir()+"/token")
		err := svc.CreateUser(context.Background(), "user", "password")

		require.NoError(t, err)
		mockAuth.AssertExpectations(t)
	})

	t.Run("already exists", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("CreateUser", mock.Anything, mock.Anything, mock.Anything).
			Return(domain.ErrAlreadyExists)

		svc := service.NewAuthService(mockAuth, t.TempDir()+"/token")
		err := svc.CreateUser(context.Background(), "user", "password")

		assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	})
}

func TestAuthService_Login(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("GetSalt", mock.Anything, "user").
			Return([]byte("saltsaltsaltsalt"), nil)
		mockAuth.On("Login", mock.Anything, mock.MatchedBy(func(creds domain.Credentials) bool {
			return creds.Login == "user" && len(creds.MasterKey) == 32
		})).Return("jwt-token", nil)

		tokenPath := t.TempDir() + "/token"
		svc := service.NewAuthService(mockAuth, tokenPath)
		err := svc.Login(context.Background(), "user", "password")

		require.NoError(t, err)

		data, err := os.ReadFile(tokenPath)
		require.NoError(t, err)
		assert.Equal(t, "jwt-token", string(data))
	})

	t.Run("get salt error", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("GetSalt", mock.Anything, "user").
			Return(nil, domain.ErrNotFound)

		svc := service.NewAuthService(mockAuth, t.TempDir()+"/token")
		err := svc.Login(context.Background(), "user", "password")

		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("GetSalt", mock.Anything, "user").
			Return([]byte("saltsaltsaltsalt"), nil)
		mockAuth.On("Login", mock.Anything, mock.Anything).
			Return("", domain.ErrInvalidCredentials)

		svc := service.NewAuthService(mockAuth, t.TempDir()+"/token")
		err := svc.Login(context.Background(), "user", "password")

		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("token saved with correct permissions", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("GetSalt", mock.Anything, "user").
			Return([]byte("saltsaltsaltsalt"), nil)
		mockAuth.On("Login", mock.Anything, mock.Anything).
			Return("jwt-token", nil)

		tokenPath := t.TempDir() + "/token"
		svc := service.NewAuthService(mockAuth, tokenPath)
		err := svc.Login(context.Background(), "user", "password")
		require.NoError(t, err)

		info, err := os.Stat(tokenPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	})
}
