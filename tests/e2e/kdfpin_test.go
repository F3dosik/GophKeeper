//go:build e2e

package e2e

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/client/grpcclient"
	"github.com/F3dosik/GophKeeper/internal/client/kdfpin"
	"github.com/F3dosik/GophKeeper/internal/client/service"
	"github.com/F3dosik/GophKeeper/internal/client/vault"
	"github.com/F3dosik/GophKeeper/internal/domain"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// weakenKDF имитирует взломанный сервер: ослабляет параметры Argon2id пользователя
// в базе, чтобы клиент вывел по ним ключ аутентификации, который дешевле перебирать.
func weakenKDF(t *testing.T, login string) {
	t.Helper()
	tag, err := db.Exec(context.Background(),
		`UPDATE users SET kdf_time = 1, kdf_memory_kib = 19456 WHERE login = $1`, login)
	require.NoError(t, err)
	require.EqualValues(t, 1, tag.RowsAffected())
}

func TestE2E_KDFPins_CLIRejectsDowngrade(t *testing.T) {
	ctx := context.Background()
	tokens := grpcclient.NewTokenStore("")
	conn, err := grpcclient.Dial(serverAddr, "", true, tokens)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session")
	auth := service.NewAuthService(grpcclient.NewAuthClient(pb.NewAuthClient(conn)), sessionPath, tokens,
		service.WithKDFPins(kdfpin.NextTo(sessionPath), serverAddr))

	login, password := "pin-"+uuid.NewString(), "pin-password-123"
	require.NoError(t, auth.CreateUser(ctx, login, password, domain.DefaultKDFParams))
	_, err = auth.Login(ctx, login, password)
	require.NoError(t, err)

	weakenKDF(t, login)

	_, err = auth.Unlock(ctx, login, password)
	var downgrade *domain.KDFDowngradeError
	require.ErrorAs(t, err, &downgrade)
	assert.Equal(t, domain.DefaultKDFParams, downgrade.Known)
	assert.Equal(t, uint32(1), downgrade.Received.Time)

	// Новое устройство (без запомненных параметров) ослабление не заметит: защита
	// работает с первого входа на устройстве. Здесь — то же устройство после подтверждения:
	// ключ выводится по новым параметрам, и сервер его не принимает (хеш остался прежним).
	require.NoError(t, auth.ForgetKDFParams(login))
	_, err = auth.Unlock(ctx, login, password)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestE2E_KDFPins_Desktop(t *testing.T) {
	app := newDesktopApp(t, t.TempDir(), &fakeUI{})
	require.NoError(t, app.SaveSettings(serverAddr, "", false, 5))

	login, password := "pin-desk-"+uuid.NewString(), "pin-password-123"
	require.NoError(t, app.Register(login, password))
	_, err := app.SignIn(login, password)
	require.NoError(t, err)
	app.Lock()

	weakenKDF(t, login)

	assert.Equal(t, vault.CodeKDFDowngrade, errorCode(t, app.Unlock(password)))
	_, err = app.SignIn(login, password)
	assert.Equal(t, vault.CodeKDFDowngrade, errorCode(t, err))

	require.NoError(t, app.AcceptKDFChange(login))
	assert.Equal(t, vault.CodeWrongPassword, errorCode(t, app.Unlock(password)))
	assert.False(t, app.GetState().Unlocked)
}
