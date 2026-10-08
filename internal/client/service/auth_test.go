package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/F3dosik/GophKeeper/pkg/crypto"
	"os"
	"testing"
	"time"

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
		}), mock.AnythingOfType("[]uint8"), domain.DefaultKDFParams, false).Return(nil, nil)

		svc := service.NewAuthService(mockAuth, t.TempDir()+"/token", nil)
		err := svc.CreateUser(context.Background(), "user", "password", domain.DefaultKDFParams)

		require.NoError(t, err)
		mockAuth.AssertExpectations(t)
	})

	t.Run("already exists", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("CreateUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything, false).
			Return(nil, domain.ErrAlreadyExists)

		svc := service.NewAuthService(mockAuth, t.TempDir()+"/token", nil)
		err := svc.CreateUser(context.Background(), "user", "password", domain.DefaultKDFParams)

		assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	})
}

func TestAuthService_Login(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("GetSalt", mock.Anything, "user").
			Return([]byte("saltsaltsaltsalt"), domain.LegacyKDFParams, nil)
		mockAuth.On("Login", mock.Anything, mock.MatchedBy(func(creds domain.Credentials) bool {
			return creds.Login == "user" && len(creds.AuthKey) == 32
		})).Return("jwt-token", false, nil)

		tokenPath := t.TempDir() + "/token"
		svc := service.NewAuthService(mockAuth, tokenPath, nil)
		_, err := svc.Login(context.Background(), "user", "password")

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
			Return(nil, domain.KDFParams{}, domain.ErrNotFound)

		svc := service.NewAuthService(mockAuth, t.TempDir()+"/token", nil)
		_, err := svc.Login(context.Background(), "user", "password")

		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("GetSalt", mock.Anything, "user").
			Return([]byte("saltsaltsaltsalt"), domain.LegacyKDFParams, nil)
		mockAuth.On("Login", mock.Anything, mock.Anything).
			Return("", false, domain.ErrInvalidCredentials)

		svc := service.NewAuthService(mockAuth, t.TempDir()+"/token", nil)
		_, err := svc.Login(context.Background(), "user", "password")

		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("token saved with correct permissions", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("GetSalt", mock.Anything, "user").
			Return([]byte("saltsaltsaltsalt"), domain.LegacyKDFParams, nil)
		mockAuth.On("Login", mock.Anything, mock.Anything).
			Return("jwt-token", false, nil)

		tokenPath := t.TempDir() + "/token"
		svc := service.NewAuthService(mockAuth, tokenPath, nil)
		_, err := svc.Login(context.Background(), "user", "password")
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
			Return([]byte("saltsaltsaltsalt"), domain.LegacyKDFParams, nil)
		mockAuth.On("Login", mock.Anything, mock.MatchedBy(func(creds domain.Credentials) bool {
			sentAuthKey = creds.AuthKey
			return creds.Login == "user" && len(creds.AuthKey) == 32
		})).Return("jwt-token", false, nil)

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
			Return([]byte("saltsaltsaltsalt"), domain.LegacyKDFParams, nil)
		mockAuth.On("Login", mock.Anything, mock.Anything).
			Return("", false, domain.ErrInvalidCredentials)

		svc := service.NewAuthService(mockAuth, t.TempDir()+"/session", nil)
		masterKey, err := svc.Unlock(context.Background(), "user", "wrong")

		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
		assert.Nil(t, masterKey)
	})

	t.Run("get salt error", func(t *testing.T) {
		mockAuth := mocks.NewAuthClient(t)
		mockAuth.On("GetSalt", mock.Anything, "user").
			Return(nil, domain.KDFParams{}, errors.New("network down"))

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

// Подменённый сервер может прислать слабые параметры Argon2id, чтобы удешевить перебор
// пароля по перехваченному authKey: клиент не должен с ними работать.
func TestAuthService_RejectsWeakServerKDF(t *testing.T) {
	mockAuth := mocks.NewAuthClient(t)
	mockAuth.On("GetSalt", mock.Anything, "user").
		Return([]byte("saltsaltsaltsalt"), domain.KDFParams{Time: 1, MemoryKiB: 8, Threads: 1}, nil)

	svc := service.NewAuthService(mockAuth, t.TempDir()+"/session", nil)

	_, err := svc.Login(context.Background(), "user", "password")
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	_, err = svc.Unlock(context.Background(), "user", "password")
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestAuthService_ChangePassword(t *testing.T) {
	salt := []byte("saltsaltsaltsalt")
	oldMaster := crypto.DeriveKey("old-password", salt, domain.LegacyKDFParams)
	oldAuth, err := crypto.HKDF(oldMaster, crypto.InfoAuth)
	require.NoError(t, err)

	var sentChange domain.PasswordChange
	reencrypted := []domain.ReencryptedSecret{{OldBlindIndex: "old", NewBlindIndex: "new", Data: []byte("x")}}

	mockAuth := mocks.NewAuthClient(t)
	mockAuth.On("GetSalt", mock.Anything, "user").Return(salt, domain.LegacyKDFParams, nil)
	mockAuth.On("ChangePassword", mock.Anything, mock.Anything, reencrypted).
		Run(func(args mock.Arguments) { sentChange = args.Get(1).(domain.PasswordChange) }).
		Return("new-token", nil)

	sessionPath := t.TempDir() + "/session"
	tokens := grpcclient.NewTokenStore("old-token")
	svc := service.NewAuthService(mockAuth, sessionPath, tokens)

	var gotNewMaster []byte
	err = svc.ChangePassword(context.Background(), "user", "old-password", "new-password", domain.DefaultKDFParams,
		func(newMasterKey []byte) ([]domain.ReencryptedSecret, error) {
			gotNewMaster = newMasterKey
			return reencrypted, nil
		})
	require.NoError(t, err)

	assert.Equal(t, oldAuth, sentChange.OldAuthKey, "proves knowledge of the current password")
	assert.Equal(t, domain.DefaultKDFParams, sentChange.NewKDF, "password change upgrades kdf params")
	assert.NotEqual(t, salt, sentChange.NewSalt, "new password gets a new salt")

	wantNewMaster := crypto.DeriveKey("new-password", sentChange.NewSalt, domain.DefaultKDFParams)
	assert.Equal(t, wantNewMaster, gotNewMaster)
	wantNewAuth, err := crypto.HKDF(wantNewMaster, crypto.InfoAuth)
	require.NoError(t, err)
	assert.Equal(t, wantNewAuth, sentChange.NewAuthKey)

	assert.Equal(t, "new-token", tokens.Token())
	sess, err := session.Load(sessionPath)
	require.NoError(t, err)
	assert.Equal(t, "new-token", sess.Token)
}

func TestAuthService_CreateTemporaryUser(t *testing.T) {
	expiresAt := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	mockAuth := mocks.NewAuthClient(t)
	mockAuth.On("CreateUser", mock.Anything, mock.Anything, mock.Anything, domain.DefaultKDFParams, true).
		Return(&expiresAt, nil)

	svc := service.NewAuthService(mockAuth, t.TempDir()+"/session", nil)
	password, gotExpiresAt, err := svc.CreateTemporaryUser(context.Background(), "bob", domain.DefaultKDFParams)
	require.NoError(t, err)

	assert.Regexp(t, `^[a-zA-Z2-9]{4}(-[a-zA-Z2-9]{4}){3}$`, password)
	assert.NotRegexp(t, `[01lIoO]`, password, "ambiguous characters must not be used")
	assert.Equal(t, expiresAt, gotExpiresAt)

	other, _, err := svc.CreateTemporaryUser(context.Background(), "bob", domain.DefaultKDFParams)
	require.NoError(t, err)
	assert.NotEqual(t, password, other)
}

func TestAuthService_Login_PasswordChangeRequired(t *testing.T) {
	mockAuth := mocks.NewAuthClient(t)
	mockAuth.On("GetSalt", mock.Anything, "bob").Return([]byte("saltsaltsaltsalt"), domain.DefaultKDFParams, nil)
	mockAuth.On("Login", mock.Anything, mock.Anything).Return("restricted", true, nil)

	sessionPath := t.TempDir() + "/session"
	tokens := grpcclient.NewTokenStore("")
	svc := service.NewAuthService(mockAuth, sessionPath, tokens)

	changeRequired, err := svc.Login(context.Background(), "bob", "temp")
	require.NoError(t, err)
	assert.True(t, changeRequired)
	assert.Equal(t, "restricted", tokens.Token(), "the token is needed for ChangePassword in this process")
	_, statErr := os.Stat(sessionPath)
	assert.True(t, os.IsNotExist(statErr), "a restricted token must not be saved to the session")

	_, err = svc.Unlock(context.Background(), "bob", "temp")
	assert.ErrorIs(t, err, domain.ErrPasswordChangeRequired)
}

// Недопустимые параметры отклоняются до Argon2id и обращения к серверу
// (мок без ожиданий упадёт при любом вызове).
func TestAuthService_RejectsInvalidKDFBeforeWork(t *testing.T) {
	svc := service.NewAuthService(mocks.NewAuthClient(t), t.TempDir()+"/session", nil)
	bad := domain.KDFParams{Time: 100, MemoryKiB: 64 * 1024, Threads: 4}

	assert.ErrorIs(t, svc.CreateUser(context.Background(), "user", "password", bad), domain.ErrInvalidArgument)
	_, _, err := svc.CreateTemporaryUser(context.Background(), "user", bad)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	err = svc.ChangePassword(context.Background(), "user", "old", "new", bad, nil)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}
