//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func credsPayload(t *testing.T, name, login, password string) *domain.SecretPayload {
	t.Helper()
	data, err := json.Marshal(domain.CredentialsSecret{Login: login, Password: password})
	require.NoError(t, err)
	return &domain.SecretPayload{
		Name: name,
		Type: domain.SecretTypeCredentials,
		Data: data,
	}
}

func TestE2E_Secret_CRUD(t *testing.T) {
	ctx := context.Background()
	kit := newClientKit(t)
	kit.registerAndLogin(ctx, t)

	payload := credsPayload(t, "github", "ivan", "s3cret")

	t.Run("create", func(t *testing.T) {
		require.NoError(t, kit.Secrets.CreateSecret(ctx, payload))
	})

	t.Run("get", func(t *testing.T) {
		got, err := kit.Secrets.GetSecret(ctx, "github", domain.SecretTypeCredentials)
		require.NoError(t, err)
		assert.Equal(t, payload.Name, got.Name)
		assert.Equal(t, payload.Type, got.Type)

		var c domain.CredentialsSecret
		require.NoError(t, json.Unmarshal(got.Data, &c))
		assert.Equal(t, "ivan", c.Login)
		assert.Equal(t, "s3cret", c.Password)
	})

	t.Run("list contains secret", func(t *testing.T) {
		list, err := kit.Secrets.ListSecrets(ctx)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, "github", list[0].Name)
	})

	t.Run("update", func(t *testing.T) {
		updated := credsPayload(t, "github", "ivan", "new-password")
		require.NoError(t, kit.Secrets.UpdateSecret(ctx, updated))

		got, err := kit.Secrets.GetSecret(ctx, "github", domain.SecretTypeCredentials)
		require.NoError(t, err)
		var c domain.CredentialsSecret
		require.NoError(t, json.Unmarshal(got.Data, &c))
		assert.Equal(t, "new-password", c.Password)
	})

	t.Run("delete", func(t *testing.T) {
		require.NoError(t, kit.Secrets.DeleteSecret(ctx, "github", domain.SecretTypeCredentials))

		_, err := kit.Secrets.GetSecret(ctx, "github", domain.SecretTypeCredentials)
		assert.Error(t, err, "secret must be gone after delete")
	})
}

func TestE2E_Secret_Isolation(t *testing.T) {
	ctx := context.Background()
	alice := newClientKit(t)
	bob := newClientKit(t)
	alice.registerAndLogin(ctx, t)
	bob.registerAndLogin(ctx, t)

	require.NoError(t, alice.Secrets.CreateSecret(ctx, credsPayload(t, "shared-name", "a", "a")))

	_, err := bob.Secrets.GetSecret(ctx, "shared-name", domain.SecretTypeCredentials)
	assert.Error(t, err, "bob must not see alice's secret even with the same name")

	list, err := bob.Secrets.ListSecrets(ctx)
	require.NoError(t, err)
	assert.Empty(t, list, "bob's list must be empty")
}
