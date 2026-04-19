//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/client/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_RegisterAndLogin(t *testing.T) {
	ctx := context.Background()
	kit := newClientKit(t)

	require.NoError(t, kit.Auth.CreateUser(ctx, kit.Login, kit.Password))
	require.NoError(t, kit.Auth.Login(ctx, kit.Login, kit.Password))

	sess, err := session.Load(kit.SessionPath)
	require.NoError(t, err)
	assert.Equal(t, kit.Login, sess.Login)
	assert.NotEmpty(t, sess.Token)
}

func TestE2E_Login_WrongPassword(t *testing.T) {
	ctx := context.Background()
	kit := newClientKit(t)

	require.NoError(t, kit.Auth.CreateUser(ctx, kit.Login, kit.Password))

	err := kit.Auth.Login(ctx, kit.Login, "wrong-password")
	assert.Error(t, err)

	_, statErr := os.Stat(kit.SessionPath)
	assert.True(t, os.IsNotExist(statErr), "session file must not be created on failed login")
}

func TestE2E_Register_DuplicateLogin(t *testing.T) {
	ctx := context.Background()
	kit := newClientKit(t)

	require.NoError(t, kit.Auth.CreateUser(ctx, kit.Login, kit.Password))
	err := kit.Auth.CreateUser(ctx, kit.Login, kit.Password)
	assert.Error(t, err)
}
