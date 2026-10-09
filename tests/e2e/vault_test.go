//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/internal/client/session"
	"github.com/F3dosik/GophKeeper/internal/client/vault"
	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lockRecorder запоминает причины блокировок, о которых сообщает Vault.
type lockRecorder struct {
	mu      sync.Mutex
	reasons []vault.LockReason
}

func (r *lockRecorder) record(reason vault.LockReason) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reasons = append(r.reasons, reason)
}

func (r *lockRecorder) last() vault.LockReason {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.reasons) == 0 {
		return ""
	}
	return r.reasons[len(r.reasons)-1]
}

// openVault открывает Vault к тестовому серверу с собственным файлом сессии.
func openVault(t *testing.T, sessionPath string, modify func(*vault.Config)) (*vault.Vault, *lockRecorder) {
	t.Helper()
	locks := &lockRecorder{}
	cfg := vault.Config{
		ServerAddress: serverAddr,
		Insecure:      true,
		SessionPath:   sessionPath,
		OnLock:        locks.record,
	}
	if modify != nil {
		modify(&cfg)
	}
	v, err := vault.Open(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = v.Close() })
	return v, locks
}

// newVaultUser регистрирует пользователя и возвращает разблокированный Vault.
func newVaultUser(t *testing.T, modify func(*vault.Config)) (*vault.Vault, *lockRecorder, string) {
	t.Helper()
	return newVaultUserAt(t, filepath.Join(t.TempDir(), "session"), modify)
}

// newVaultUserAt — как newVaultUser, но с заданным файлом сессии.
func newVaultUserAt(t *testing.T, sessionPath string, modify func(*vault.Config)) (*vault.Vault, *lockRecorder, string) {
	t.Helper()
	ctx := context.Background()
	v, locks := openVault(t, sessionPath, modify)
	login := fmt.Sprintf("vault-%s", uuid.NewString())
	require.NoError(t, v.Register(ctx, login, "vault-password-1", domain.DefaultKDFParams))
	changeRequired, err := v.SignIn(ctx, login, "vault-password-1")
	require.NoError(t, err)
	require.False(t, changeRequired)
	return v, locks, login
}

// readToken возвращает токен из файла сессии.
func readToken(t *testing.T, sessionPath string) string {
	t.Helper()
	sess, err := session.Load(sessionPath)
	require.NoError(t, err)
	return sess.Token
}

func textPayload(name, text string) *domain.SecretPayload {
	data, _ := json.Marshal(domain.TextSecret{Text: text})
	return &domain.SecretPayload{Name: name, Type: domain.SecretTypeText, Data: data}
}

func TestE2E_Vault_LockUnlock(t *testing.T) {
	ctx := context.Background()
	v, locks, _ := newVaultUser(t, nil)

	require.NoError(t, v.CreateSecret(ctx, textPayload("note", "hello")))
	list, err := v.ListSecrets(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "note", list[0].Name)

	v.Lock()
	assert.False(t, v.IsUnlocked())
	assert.Equal(t, vault.LockManual, locks.last())
	_, err = v.ListSecrets(ctx)
	assert.Equal(t, vault.CodeLocked, vault.ErrorCode(err))

	err = v.Unlock(ctx, "wrong-password-1")
	assert.Equal(t, vault.CodeWrongPassword, vault.ErrorCode(err))
	assert.False(t, v.IsUnlocked())

	require.NoError(t, v.Unlock(ctx, "vault-password-1"))
	info, err := v.GetSecret(ctx, "note", domain.SecretTypeText)
	require.NoError(t, err)
	assert.JSONEq(t, `{"text":"hello"}`, string(info.Data))
}

func TestE2E_Vault_RestoresSessionOnOpen(t *testing.T) {
	ctx := context.Background()
	sessionPath := filepath.Join(t.TempDir(), "session")
	first, _ := openVault(t, sessionPath, nil)
	login := fmt.Sprintf("vault-%s", uuid.NewString())
	require.NoError(t, first.Register(ctx, login, "vault-password-1", domain.DefaultKDFParams))
	_, err := first.SignIn(ctx, login, "vault-password-1")
	require.NoError(t, err)
	require.NoError(t, first.Close())

	// Перезапуск приложения: логин помнится, но ключей нет — нужен пароль.
	second, _ := openVault(t, sessionPath, nil)
	assert.Equal(t, login, second.CurrentLogin())
	assert.False(t, second.IsUnlocked())
	require.NoError(t, second.Unlock(ctx, "vault-password-1"))
	assert.True(t, second.IsUnlocked())
}

func TestE2E_Vault_AutoLock(t *testing.T) {
	ctx := context.Background()
	v, locks, _ := newVaultUser(t, func(c *vault.Config) { c.AutoLockAfter = 300 * time.Millisecond })

	// Активность откладывает блокировку.
	for range 3 {
		time.Sleep(150 * time.Millisecond)
		_, err := v.ListSecrets(ctx)
		require.NoError(t, err)
	}
	assert.True(t, v.IsUnlocked())

	assert.Eventually(t, func() bool { return !v.IsUnlocked() }, 2*time.Second, 20*time.Millisecond)
	assert.Equal(t, vault.LockIdle, locks.last())
}

func TestE2E_Vault_RefreshesTokenAfterSuccess(t *testing.T) {
	ctx := context.Background()
	// Запас больше срока жизни токена: обновление после каждого успешного запроса.
	sessionPath := filepath.Join(t.TempDir(), "session")
	v, _, _ := newVaultUserAt(t, sessionPath, func(c *vault.Config) { c.RefreshMargin = 2 * time.Hour })

	before := readToken(t, sessionPath)
	_, err := v.ListSecrets(ctx)
	require.NoError(t, err)
	assert.NotEqual(t, before, readToken(t, sessionPath), "token must be refreshed without a password")
}

// Выход на всех устройствах с другого клиента должен блокировать и разблокированное
// приложение: Vault не продлевает отозванный токен.
func TestE2E_Vault_LogoutEverywhereLocksOtherDevice(t *testing.T) {
	ctx := context.Background()
	desktop, locks, login := newVaultUser(t, func(c *vault.Config) { c.RefreshMargin = 2 * time.Hour })

	phone, _ := openVault(t, filepath.Join(t.TempDir(), "session"), nil)
	_, err := phone.SignIn(ctx, login, "vault-password-1")
	require.NoError(t, err)
	require.NoError(t, phone.Logout(ctx, true))

	_, err = desktop.ListSecrets(ctx)
	assert.Equal(t, vault.CodeSessionExpired, vault.ErrorCode(err))
	assert.False(t, desktop.IsUnlocked())
	assert.Equal(t, vault.LockSessionExpired, locks.last())

	require.NoError(t, desktop.Unlock(ctx, "vault-password-1"), "the password still works")
}

func TestE2E_Vault_ChangePassword(t *testing.T) {
	ctx := context.Background()
	v, _, _ := newVaultUser(t, nil)
	require.NoError(t, v.CreateSecret(ctx, textPayload("note", "kept")))

	err := v.ChangePassword(ctx, "wrong-password-1", "vault-password-2", domain.DefaultKDFParams)
	assert.Equal(t, vault.CodeWrongPassword, vault.ErrorCode(err))
	assert.True(t, v.IsUnlocked(), "a wrong current password must not lock the vault")

	require.NoError(t, v.ChangePassword(ctx, "vault-password-1", "vault-password-2", domain.DefaultKDFParams))
	assert.True(t, v.IsUnlocked())
	info, err := v.GetSecret(ctx, "note", domain.SecretTypeText)
	require.NoError(t, err)
	assert.JSONEq(t, `{"text":"kept"}`, string(info.Data))

	v.Lock()
	assert.Equal(t, vault.CodeWrongPassword, vault.ErrorCode(v.Unlock(ctx, "vault-password-1")))
	assert.NoError(t, v.Unlock(ctx, "vault-password-2"))
}

func TestE2E_Vault_TemporaryPassword(t *testing.T) {
	ctx := context.Background()
	admin := newClientKit(t)
	tempPassword, _, err := admin.Auth.CreateTemporaryUser(ctx, admin.Login, domain.DefaultKDFParams)
	require.NoError(t, err)

	v, _ := openVault(t, filepath.Join(t.TempDir(), "session"), nil)
	changeRequired, err := v.SignIn(ctx, admin.Login, tempPassword)
	require.NoError(t, err)
	require.True(t, changeRequired)
	assert.False(t, v.IsUnlocked())

	require.NoError(t, v.CompletePasswordChange(ctx, tempPassword, "own-password-123"))
	assert.True(t, v.IsUnlocked())
	require.NoError(t, v.CreateSecret(ctx, textPayload("first", "x")))
}

func TestE2E_Vault_DeleteAccount(t *testing.T) {
	ctx := context.Background()
	v, locks, _ := newVaultUser(t, nil)

	assert.Equal(t, vault.CodeWrongPassword, vault.ErrorCode(v.DeleteAccount(ctx, "wrong-password-1")))
	require.NoError(t, v.DeleteAccount(ctx, "vault-password-1"))

	assert.Empty(t, v.CurrentLogin())
	assert.Equal(t, vault.LockSignedOut, locks.last())
	assert.Equal(t, vault.CodeNotSignedIn, vault.ErrorCode(v.Unlock(ctx, "vault-password-1")))
}
