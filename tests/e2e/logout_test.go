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

// listWithToken выполняет ListSecrets на отдельном соединении с заданным токеном,
// как это сделал бы злоумышленник, укравший файл сессии.
func listWithToken(ctx context.Context, t *testing.T, token string) error {
	t.Helper()

	conn, err := grpcclient.Dial(serverAddr, "", true, grpcclient.NewTokenStore(token))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	_, err = grpcclient.NewSecretsClient(pb.NewSecretsClient(conn)).ListSecrets(ctx)
	return err
}

func TestE2E_Logout_RevokesCurrentToken(t *testing.T) {
	ctx := context.Background()
	kit := newClientKit(t)
	kit.registerAndLogin(ctx, t)

	stolen := kit.Tokens.Token()
	require.NoError(t, listWithToken(ctx, t, stolen), "token works before logout")

	require.NoError(t, kit.Auth.Logout(ctx, false))

	err := listWithToken(ctx, t, stolen)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials, "token must be rejected after logout")
}

func TestE2E_Logout_OtherDeviceUnaffected(t *testing.T) {
	ctx := context.Background()
	laptop := newClientKit(t)
	laptop.registerAndLogin(ctx, t)

	phone := newClientKit(t)
	phone.Login, phone.Password = laptop.Login, laptop.Password
	require.NoError(t, phone.Auth.Login(ctx, phone.Login, phone.Password))

	require.NoError(t, laptop.Auth.Logout(ctx, false))

	assert.NoError(t, listWithToken(ctx, t, phone.Tokens.Token()))
}

func TestE2E_Logout_AllSessions(t *testing.T) {
	ctx := context.Background()
	laptop := newClientKit(t)
	laptop.registerAndLogin(ctx, t)

	phone := newClientKit(t)
	phone.Login, phone.Password = laptop.Login, laptop.Password
	require.NoError(t, phone.Auth.Login(ctx, phone.Login, phone.Password))
	phoneToken := phone.Tokens.Token()

	require.NoError(t, laptop.Auth.Logout(ctx, true))

	err := listWithToken(ctx, t, phoneToken)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials, "all tokens must be revoked")

	// После выхода на всех устройствах можно снова войти, новый токен действителен.
	require.NoError(t, phone.Auth.Login(ctx, phone.Login, phone.Password))
	assert.NoError(t, listWithToken(ctx, t, phone.Tokens.Token()))
}

func TestE2E_Unlock_RefreshesToken(t *testing.T) {
	ctx := context.Background()
	kit := newClientKit(t)
	kit.registerAndLogin(ctx, t)
	before := kit.Tokens.Token()

	_, err := kit.Auth.Unlock(ctx, kit.Login, kit.Password)
	require.NoError(t, err)

	assert.NotEqual(t, before, kit.Tokens.Token(), "Unlock must issue a fresh token")
	assert.NoError(t, listWithToken(ctx, t, kit.Tokens.Token()))
}
