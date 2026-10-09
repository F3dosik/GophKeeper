//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/client/grpcclient"
	"github.com/F3dosik/GophKeeper/internal/domain"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findUser(users []*domain.UserInfo, login string) *domain.UserInfo {
	for _, u := range users {
		if u.Login == login {
			return u
		}
	}
	return nil
}

func TestE2E_Admin_UsersOverview(t *testing.T) {
	ctx := context.Background()
	admin := grpcclient.NewAdminClient(pb.NewAdminClient(newClientKit(t).conn))

	// Обычный пользователь с двумя секретами.
	alice := newClientKit(t)
	alice.registerAndLogin(ctx, t)
	alice.initSecretsService(ctx, t)
	require.NoError(t, alice.Secrets.CreateSecret(ctx, credsPayload(t, "a", "u", "p")))
	require.NoError(t, alice.Secrets.CreateSecret(ctx, credsPayload(t, "b", "u", "p")))

	// Пользователь с временным паролем.
	bob := newClientKit(t)
	_, _, err := bob.Auth.CreateTemporaryUser(ctx, bob.Login, domain.DefaultKDFParams)
	require.NoError(t, err)

	users, err := admin.ListUsers(ctx)
	require.NoError(t, err)

	a := findUser(users, alice.Login)
	require.NotNil(t, a)
	assert.Equal(t, 2, a.SecretCount)
	assert.Nil(t, a.PasswordExpiresAt)
	assert.Equal(t, domain.DefaultKDFParams.Time, a.KDF.Time)

	b := findUser(users, bob.Login)
	require.NotNil(t, b)
	assert.NotNil(t, b.PasswordExpiresAt, "temporary password is visible to the admin")

	info, err := admin.GetUser(ctx, alice.Login)
	require.NoError(t, err)
	assert.Equal(t, 2, info.SecretCount)

	_, err = admin.GetUser(ctx, "no-such-user")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestE2E_Admin_RevokeSessions(t *testing.T) {
	ctx := context.Background()
	admin := grpcclient.NewAdminClient(pb.NewAdminClient(newClientKit(t).conn))

	user := newClientKit(t)
	user.registerAndLogin(ctx, t)
	token := user.Tokens.Token()
	require.NoError(t, listWithToken(ctx, t, token))

	require.NoError(t, admin.RevokeSessions(ctx, user.Login))
	assert.ErrorIs(t, listWithToken(ctx, t, token), domain.ErrInvalidCredentials, "token must be revoked")

	// Пароль не меняется: пользователь просто входит заново.
	_, err := user.Auth.Login(ctx, user.Login, user.Password)
	require.NoError(t, err)
	assert.NoError(t, listWithToken(ctx, t, user.Tokens.Token()))

	assert.ErrorIs(t, admin.RevokeSessions(ctx, "no-such-user"), domain.ErrNotFound)
}
