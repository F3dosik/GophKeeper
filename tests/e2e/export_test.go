//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/internal/client/export"
	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func payloadsOf(t *testing.T, infos []*domain.SecretInfo) []domain.SecretPayload {
	t.Helper()
	result := make([]domain.SecretPayload, 0, len(infos))
	for _, info := range infos {
		result = append(result, info.SecretPayload)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func TestE2E_ExportImport_RoundTrip(t *testing.T) {
	ctx := context.Background()
	src := newClientKit(t)
	src.registerAndLogin(ctx, t)
	src.initSecretsService(ctx, t)

	card, _ := json.Marshal(domain.CardSecret{Number: "4111111111111111", Holder: "IVAN", Expiry: "12/30", CVV: "123"})
	file, _ := json.Marshal(domain.BinarySecret{Data: []byte{0, 1, 2, 255}})
	for _, p := range []*domain.SecretPayload{
		credsPayload(t, "github", "me", "p@ss"),
		textPayload("note", "многострочный\nтекст"),
		{Name: "visa", Type: domain.SecretTypeCard, Data: card, Metadata: "основная"},
		{Name: "key.bin", Type: domain.SecretTypeBinary, Data: file},
	} {
		require.NoError(t, src.Secrets.CreateSecret(ctx, p))
	}

	infos, err := src.Secrets.ListSecrets(ctx)
	require.NoError(t, err)
	exported := payloadsOf(t, infos)

	data, err := export.Encrypt(&export.Archive{ExportedAt: time.Now(), Login: src.Login, Secrets: exported}, "export-pass-123")
	require.NoError(t, err)

	// Импорт в другую учётку.
	dst := newClientKit(t)
	dst.registerAndLogin(ctx, t)
	dst.initSecretsService(ctx, t)

	archive, err := export.Decrypt(data, "export-pass-123")
	require.NoError(t, err)
	report, err := export.Import(ctx, dst.Secrets, archive, false, nil)
	require.NoError(t, err)
	assert.Equal(t, 4, report.Created)
	assert.Empty(t, report.Failed)

	imported, err := dst.Secrets.ListSecrets(ctx)
	require.NoError(t, err)
	got := payloadsOf(t, imported)
	require.Len(t, got, 4)
	for i := range exported {
		assert.Equal(t, exported[i].Name, got[i].Name)
		assert.Equal(t, exported[i].Type, got[i].Type)
		assert.Equal(t, exported[i].Metadata, got[i].Metadata)
		assert.JSONEq(t, string(exported[i].Data), string(got[i].Data), exported[i].Name)
	}

	// Повторный импорт без overwrite ничего не дублирует.
	report, err = export.Import(ctx, dst.Secrets, archive, false, nil)
	require.NoError(t, err)
	assert.Equal(t, 4, report.Skipped)
	assert.Zero(t, report.Created)
}

func TestE2E_Vault_ExportSecrets(t *testing.T) {
	ctx := context.Background()
	v, _, _ := newVaultUserAt(t, filepath.Join(t.TempDir(), "session"), nil)
	require.NoError(t, v.CreateSecret(ctx, textPayload("a", "1")))
	require.NoError(t, v.CreateSecret(ctx, textPayload("b", "2")))

	secrets, err := v.ExportSecrets(ctx)
	require.NoError(t, err)
	assert.Len(t, secrets, 2)

	v.Lock()
	_, err = v.ExportSecrets(ctx)
	assert.Error(t, err, "export needs an unlocked vault")
}
