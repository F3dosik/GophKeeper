//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/internal/client/session"
	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Администратор создаёт учётку с временным паролем, пользователь входит им,
// обязан сменить пароль и только после этого получает обычный доступ.
func TestE2E_TemporaryPassword_ForcedChange(t *testing.T) {
	ctx := context.Background()

	admin := newClientKit(t)
	tempPassword, expiresAt, err := admin.Auth.CreateTemporaryUser(ctx, admin.Login, domain.DefaultKDFParams)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(time.Hour), expiresAt, time.Minute)

	user := newClientKit(t)
	user.Login = admin.Login

	changeRequired, err := user.Auth.Login(ctx, user.Login, tempPassword)
	require.NoError(t, err)
	require.True(t, changeRequired)

	// Токен временного пароля не даёт доступа к секретам и не сохраняется в сессию.
	assert.ErrorIs(t, listWithToken(ctx, t, user.Tokens.Token()), domain.ErrPermissionDenied)
	_, err = session.Load(user.SessionPath)
	assert.Error(t, err)

	_, err = user.Auth.Unlock(ctx, user.Login, tempPassword)
	assert.ErrorIs(t, err, domain.ErrPasswordChangeRequired)

	require.NoError(t, user.Auth.ChangePassword(ctx, user.Login, tempPassword, newPassword, domain.DefaultKDFParams, nil))

	// Новый пароль даёт полный доступ, временный больше не подходит.
	assert.NoError(t, listWithToken(ctx, t, user.Tokens.Token()))
	changeRequired, err = user.Auth.Login(ctx, user.Login, newPassword)
	require.NoError(t, err)
	assert.False(t, changeRequired)
	_, err = user.Auth.Login(ctx, user.Login, tempPassword)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)

	// Новые учётки и сменившие пароль получают актуальные параметры Argon2id.
	user.Password = newPassword
	user.initSecretsService(ctx, t)
	require.NoError(t, user.Secrets.CreateSecret(ctx, credsPayload(t, "site", "u", "p")))
}
