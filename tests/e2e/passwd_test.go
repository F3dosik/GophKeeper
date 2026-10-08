//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/client/grpcclient"
	"github.com/F3dosik/GophKeeper/internal/client/service"
	"github.com/F3dosik/GophKeeper/internal/domain"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const newPassword = "brand-new-password-456"

// secretsFor возвращает SecretsService для пароля password на соединении kit.
func (k *clientKit) secretsFor(ctx context.Context, t *testing.T, password string) service.SecretsService {
	t.Helper()
	masterKey, err := k.Auth.Unlock(ctx, k.Login, password)
	require.NoError(t, err)
	svc, err := service.NewSecretsService(grpcclient.NewSecretsClient(pb.NewSecretsClient(k.conn)), masterKey)
	require.NoError(t, err)
	return svc
}

func TestE2E_ChangePassword_ReencryptsSecrets(t *testing.T) {
	ctx := context.Background()
	kit := newClientKit(t)
	kit.registerAndLogin(ctx, t)
	kit.initSecretsService(ctx, t)

	for _, name := range []string{"github", "bank", "mail"} {
		require.NoError(t, kit.Secrets.CreateSecret(ctx, credsPayload(t, name, "user-"+name, "pass-"+name)))
	}

	require.NoError(t, kit.Auth.ChangePassword(ctx, kit.Login, kit.Password, newPassword,
		func(newMasterKey []byte) ([]domain.ReencryptedSecret, error) {
			return kit.Secrets.Reencrypt(ctx, newMasterKey)
		}))

	_, err := kit.Auth.Unlock(ctx, kit.Login, kit.Password)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials, "old password must stop working")

	secrets := kit.secretsFor(ctx, t, newPassword)
	list, err := secrets.ListSecrets(ctx)
	require.NoError(t, err)
	require.Len(t, list, 3)

	got, err := secrets.GetSecret(ctx, "bank", domain.SecretTypeCredentials)
	require.NoError(t, err)
	var creds domain.CredentialsSecret
	require.NoError(t, json.Unmarshal(got.Data, &creds))
	assert.Equal(t, "pass-bank", creds.Password)
}

func TestE2E_ChangePassword_RevokesOtherDevices(t *testing.T) {
	ctx := context.Background()
	laptop := newClientKit(t)
	laptop.registerAndLogin(ctx, t)

	phone := newClientKit(t)
	phone.Login, phone.Password = laptop.Login, laptop.Password
	_, err := phone.Auth.Login(ctx, phone.Login, phone.Password)
	require.NoError(t, err)
	phoneToken := phone.Tokens.Token()

	require.NoError(t, laptop.Auth.ChangePassword(ctx, laptop.Login, laptop.Password, newPassword, nil))

	assert.ErrorIs(t, listWithToken(ctx, t, phoneToken), domain.ErrInvalidCredentials)
	assert.NoError(t, listWithToken(ctx, t, laptop.Tokens.Token()), "the changing device gets a fresh token")
}

// Секрет, созданный между перешифровкой и отправкой, не попал бы в новый набор и остался
// бы со старыми ключами. Сервер обязан отменить смену целиком.
func TestE2E_ChangePassword_AbortsWhenSecretsChange(t *testing.T) {
	ctx := context.Background()
	kit := newClientKit(t)
	kit.registerAndLogin(ctx, t)
	kit.initSecretsService(ctx, t)
	require.NoError(t, kit.Secrets.CreateSecret(ctx, credsPayload(t, "github", "u", "p")))

	err := kit.Auth.ChangePassword(ctx, kit.Login, kit.Password, newPassword,
		func(newMasterKey []byte) ([]domain.ReencryptedSecret, error) {
			items, err := kit.Secrets.Reencrypt(ctx, newMasterKey)
			if err != nil {
				return nil, err
			}
			// «Другое устройство» добавляет секрет, пока идёт смена пароля.
			return items, kit.Secrets.CreateSecret(ctx, credsPayload(t, "late", "u", "p"))
		})
	assert.ErrorIs(t, err, domain.ErrSecretsChanged)

	// Ничего не изменилось: старый пароль работает, оба секрета на месте и читаются.
	secrets := kit.secretsFor(ctx, t, kit.Password)
	list, err := secrets.ListSecrets(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestE2E_ChangePassword_WrongCurrentPassword(t *testing.T) {
	ctx := context.Background()
	kit := newClientKit(t)
	kit.registerAndLogin(ctx, t)

	err := kit.Auth.ChangePassword(ctx, kit.Login, "wrong-password-1", newPassword, nil)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)

	_, err = kit.Auth.Unlock(ctx, kit.Login, kit.Password)
	assert.NoError(t, err, "password must stay unchanged")
}
