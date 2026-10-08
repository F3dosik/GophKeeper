package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/client/grpcclient"
	"github.com/F3dosik/GophKeeper/internal/client/mocks"
	"github.com/F3dosik/GophKeeper/internal/client/service"
	"github.com/F3dosik/GophKeeper/internal/client/session"
	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAuthService_CreateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("CreateUser", mock.Anything, mock.MatchedBy(func(creds domain.Credentials) bool {
			return creds.Login == "user" && len(creds.AuthKey) == 32
		}), mock.AnythingOfType("[]uint8")).Return(nil)

		svc := service.NewAuthService(mockAuth, t.TempDir()+"/token", nil)
		err := svc.CreateUser(context.Background(), "user", "password")

		require.NoError(t, err)
		mockAuth.AssertExpectations(t)
	})

	t.Run("already exists", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("CreateUser", mock.Anything, mock.Anything, mock.Anything).
			Return(domain.ErrAlreadyExists)

		svc := service.NewAuthService(mockAuth, t.TempDir()+"/token", nil)
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
			return creds.Login == "user" && len(creds.AuthKey) == 32
		})).Return("jwt-token", nil)

		tokenPath := t.TempDir() + "/token"
		svc := service.NewAuthService(mockAuth, tokenPath, nil)
		err := svc.Login(context.Background(), "user", "password")

		require.NoError(t, err)

		data, err := os.ReadFile(tokenPath)
		require.NoError(t, err)
		expect, err := json.Marshal(session.Session{Login: "user", Token: "jwt-token"})
		require.NoError(t, err)
		assert.Equal(t, expect, data)
	})

	t.Run("get salt error", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("GetSalt", mock.Anything, "user").
			Return(nil, domain.ErrNotFound)

		svc := service.NewAuthService(mockAuth, t.TempDir()+"/token", nil)
		err := svc.Login(context.Background(), "user", "password")

		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("GetSalt", mock.Anything, "user").
			Return([]byte("saltsaltsaltsalt"), nil)
		mockAuth.On("Login", mock.Anything, mock.Anything).
			Return("", domain.ErrInvalidCredentials)

		svc := service.NewAuthService(mockAuth, t.TempDir()+"/token", nil)
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
		svc := service.NewAuthService(mockAuth, tokenPath, nil)
		err := svc.Login(context.Background(), "user", "password")
		require.NoError(t, err)

		info, err := os.Stat(tokenPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	})
}

func TestAuthService_Unlock(t *testing.T) {
	t.Run("correct password returns master key", func(t *testing.T) {
		var sentAuthKey []byte
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("GetSalt", mock.Anything, "user").
			Return([]byte("saltsaltsaltsalt"), nil)
		mockAuth.On("Login", mock.Anything, mock.MatchedBy(func(creds domain.Credentials) bool {
			sentAuthKey = creds.AuthKey
			return creds.Login == "user" && len(creds.AuthKey) == 32
		})).Return("jwt-token", nil)

		sessionPath := t.TempDir() + "/session"
		tokens := grpcclient.NewTokenStore("old-token")
		svc := service.NewAuthService(mockAuth, sessionPath, tokens)
		masterKey, err := svc.Unlock(context.Background(), "user", "password")

		require.NoError(t, err)
		assert.Len(t, masterKey, 32)
		assert.NotEqual(t, sentAuthKey, masterKey, "master key must never be sent to the server")

		// Новый токен сразу используется соединением и сохраняется для следующих запусков.
		assert.Equal(t, "jwt-token", tokens.Token())
		sess, err := session.Load(sessionPath)
		require.NoError(t, err)
		assert.Equal(t, "jwt-token", sess.Token)
	})

	t.Run("wrong password", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("GetSalt", mock.Anything, "user").
			Return([]byte("saltsaltsaltsalt"), nil)
		mockAuth.On("Login", mock.Anything, mock.Anything).
			Return("", domain.ErrInvalidCredentials)

		svc := service.NewAuthService(mockAuth, t.TempDir()+"/session", nil)
		masterKey, err := svc.Unlock(context.Background(), "user", "wrong")

		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
		assert.Nil(t, masterKey)
	})

	t.Run("get salt error", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("GetSalt", mock.Anything, "user").
			Return(nil, errors.New("network down"))

		svc := service.NewAuthService(mockAuth, t.TempDir()+"/session", nil)
		_, err := svc.Unlock(context.Background(), "user", "password")

		assert.Error(t, err)
	})
}

func TestAuthService_Logout(t *testing.T) {
	newSession := func(t *testing.T) string {
		t.Helper()
		path := t.TempDir() + "/session"
		require.NoError(t, session.Save(path, &session.Session{Login: "user", Token: "jwt-token"}))
		return path
	}

	t.Run("revokes token and removes session", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("Logout", mock.Anything, false).Return(nil)

		path := newSession(t)
		tokens := grpcclient.NewTokenStore("jwt-token")
		svc := service.NewAuthService(mockAuth, path, tokens)

		require.NoError(t, svc.Logout(context.Background(), false))
		_, err := os.Stat(path)
		assert.True(t, os.IsNotExist(err))
		assert.Empty(t, tokens.Token())
	})

	t.Run("expired token still removes local session", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("Logout", mock.Anything, false).Return(domain.ErrInvalidCredentials)

		path := newSession(t)
		svc := service.NewAuthService(mockAuth, path, nil)

		require.NoError(t, svc.Logout(context.Background(), false))
		_, err := os.Stat(path)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("all sessions with expired token keeps session", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("Logout", mock.Anything, true).Return(domain.ErrInvalidCredentials)

		path := newSession(t)
		svc := service.NewAuthService(mockAuth, path, nil)

		err := svc.Logout(context.Background(), true)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
		_, statErr := os.Stat(path)
		assert.NoError(t, statErr, "other devices were not logged out, so the failure must be visible")
	})

	t.Run("server error keeps session", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("Logout", mock.Anything, false).Return(errors.New("network down"))

		path := newSession(t)
		svc := service.NewAuthService(mockAuth, path, nil)

		assert.Error(t, svc.Logout(context.Background(), false))
		_, statErr := os.Stat(path)
		assert.NoError(t, statErr, "token was not revoked, so the user must be able to retry")
	})
}
