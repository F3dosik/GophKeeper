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

func TestE2E_DeleteAccount(t *testing.T) {
	ctx := context.Background()
	kit := newClientKit(t)
	kit.registerAndLogin(ctx, t)
	kit.initSecretsService(ctx, t)
	require.NoError(t, kit.Secrets.CreateSecret(ctx, credsPayload(t, "github", "u", "p")))
	token := kit.Tokens.Token()

	// Неверный пароль: ничего не удаляется, даже при действующем токене.
	err := kit.Auth.DeleteAccount(ctx, kit.Login, "wrong-password-1")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	assert.NoError(t, listWithToken(ctx, t, token))

	require.NoError(t, kit.Auth.DeleteAccount(ctx, kit.Login, kit.Password))

	assert.ErrorIs(t, listWithToken(ctx, t, token), domain.ErrInvalidCredentials, "tokens stop working")
	_, err = kit.Auth.Login(ctx, kit.Login, kit.Password)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)

	// Логин освобождается, а секреты удалены вместе с учёткой.
	again := newClientKit(t)
	again.Login, again.Password = kit.Login, "another-password-9"
	again.registerAndLogin(ctx, t)
	again.initSecretsService(ctx, t)
	list, err := again.Secrets.ListSecrets(ctx)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestE2E_AdminDeleteUser(t *testing.T) {
	ctx := context.Background()
	victim := newClientKit(t)
	victim.registerAndLogin(ctx, t)
	token := victim.Tokens.Token()

	admin := grpcclient.NewAdminClient(pb.NewAdminClient(newClientKit(t).conn))
	require.NoError(t, admin.DeleteUser(ctx, victim.Login))

	assert.ErrorIs(t, listWithToken(ctx, t, token), domain.ErrInvalidCredentials)
	assert.ErrorIs(t, admin.DeleteUser(ctx, victim.Login), domain.ErrNotFound)
}
