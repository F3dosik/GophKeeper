//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/desktop/backend"
	"github.com/F3dosik/GophKeeper/internal/client/vault"
	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeUI подменяет оболочку Wails: диалоги возвращают заданные пути, буфер обмена
// хранится в памяти, события записываются.
type fakeUI struct {
	mu        sync.Mutex
	openPath  string
	savePath  string
	clipboard string
	events    []string
}

func (u *fakeUI) OpenFile(string, []backend.FileFilter) (string, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.openPath, nil
}

func (u *fakeUI) SaveFile(string, string) (string, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.savePath, nil
}

func (u *fakeUI) ClipboardSet(text string) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.clipboard = text
	return nil
}

func (u *fakeUI) ClipboardGet() (string, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.clipboard, nil
}

func (u *fakeUI) Emit(event string, data ...any) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.events = append(u.events, fmt.Sprintf("%s %v", event, data))
}

func (u *fakeUI) clip() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.clipboard
}

func (u *fakeUI) lastEvent() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.events) == 0 {
		return ""
	}
	return u.events[len(u.events)-1]
}

// newDesktopApp создаёт App с каталогом данных dir.
func newDesktopApp(t *testing.T, dir string, ui *fakeUI) *backend.App {
	t.Helper()
	app := backend.NewApp(backend.Options{Dir: dir, UI: ui, Insecure: true, ClipboardClearAfter: 100 * time.Millisecond})
	require.NoError(t, backend.Startup(app, context.Background()))
	t.Cleanup(func() { backend.Shutdown(app) })
	return app
}

// errorCode достаёт код из ошибки, как это делает интерфейс.
func errorCode(t *testing.T, err error) vault.Code {
	t.Helper()
	require.Error(t, err)
	var uiErr backend.Error
	require.NoError(t, json.Unmarshal([]byte(err.Error()), &uiErr), "error must be JSON: %s", err)
	assert.NotEmpty(t, uiErr.Message)
	return uiErr.Code
}

func TestE2E_Desktop_FullFlow(t *testing.T) {
	dir := t.TempDir()
	ui := &fakeUI{}
	app := newDesktopApp(t, dir, ui)

	// Первый запуск: сервер не задан.
	assert.False(t, app.GetState().Configured)
	_, err := app.SignIn("x", "y")
	assert.Equal(t, backend.CodeNotConfigured, errorCode(t, err))

	assert.Equal(t, backend.CodeValidation, errorCode(t, app.SaveSettings("no-port", "", 5)))
	msg, err := app.CheckServer(serverAddr, "")
	require.NoError(t, err)
	assert.Contains(t, msg, "Сервер доступен")
	require.NoError(t, app.SaveSettings(serverAddr, "", 5))
	assert.True(t, app.GetState().Configured)

	// Регистрация и вход.
	login := "desk-" + uuid.NewString()
	assert.Equal(t, backend.CodeValidation, errorCode(t, app.Register(login, "short")))
	require.NoError(t, app.Register(login, "desktop-pass-1"))
	changeRequired, err := app.SignIn(login, "desktop-pass-1")
	require.NoError(t, err)
	assert.False(t, changeRequired)
	state := app.GetState()
	assert.Equal(t, login, state.Login)
	assert.True(t, state.Unlocked)

	// Секреты всех типов.
	require.NoError(t, app.SaveSecret(backend.SecretInput{Name: "github", Type: "credentials", Login: "me", Password: "pw-1"}, false))
	require.NoError(t, app.SaveSecret(backend.SecretInput{Name: "note", Type: "text", Text: "hello", Metadata: "personal"}, false))
	require.NoError(t, app.SaveSecret(backend.SecretInput{
		Name: "visa", Type: "card", CardNumber: "4111 1111 1111 1111", CardHolder: "IVAN IVANOV", CardExpiry: "12/30", CardCVV: "123",
	}, false))

	assert.Equal(t, backend.CodeValidation, errorCode(t, app.SaveSecret(backend.SecretInput{
		Name: "bad", Type: "card", CardNumber: "1234", CardHolder: "X", CardExpiry: "12/30", CardCVV: "123",
	}, false)), "invalid card number")
	assert.Equal(t, backend.CodeValidation, errorCode(t, app.SaveSecret(backend.SecretInput{Type: "text", Text: "x"}, false)))
	assert.Equal(t, vault.CodeAlreadyExists, errorCode(t, app.SaveSecret(backend.SecretInput{Name: "note", Type: "text", Text: "dup"}, false)))

	// Файл: выбор, сохранение, выгрузка.
	src := filepath.Join(t.TempDir(), "key.bin")
	require.NoError(t, os.WriteFile(src, []byte{0, 1, 2, 3, 255}, 0o600))
	ui.openPath = src
	chosen, err := app.ChooseFile()
	require.NoError(t, err)
	assert.Equal(t, int64(5), chosen.Size)
	require.NoError(t, app.SaveSecret(backend.SecretInput{Name: "key.bin", Type: "binary", FilePath: chosen.Path}, false))

	list, err := app.ListSecrets()
	require.NoError(t, err)
	assert.Len(t, list, 4)

	card, err := app.GetSecret("visa", "card")
	require.NoError(t, err)
	assert.Equal(t, "IVAN IVANOV", card.CardHolder)
	file, err := app.GetSecret("key.bin", "binary")
	require.NoError(t, err)
	assert.Equal(t, 5, file.FileSize)

	dst := filepath.Join(t.TempDir(), "restored.bin")
	ui.savePath = dst
	path, err := app.ExportFile("key.bin")
	require.NoError(t, err)
	assert.Equal(t, dst, path)
	content, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, []byte{0, 1, 2, 3, 255}, content)
	stat, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), stat.Mode().Perm())

	// Изменение файла без нового содержимого сохраняет прежнее, меняя метаданные.
	require.NoError(t, app.SaveSecret(backend.SecretInput{Name: "key.bin", Type: "binary", Metadata: "ssh"}, true))
	file, err = app.GetSecret("key.bin", "binary")
	require.NoError(t, err)
	assert.Equal(t, 5, file.FileSize)
	assert.Equal(t, "ssh", file.Metadata)

	// Блокировка.
	app.Lock()
	assert.False(t, app.GetState().Unlocked)
	assert.Equal(t, "vault:locked [manual]", ui.lastEvent())
	_, err = app.GetSecret("github", "credentials")
	assert.Equal(t, vault.CodeLocked, errorCode(t, err))
	assert.Equal(t, vault.CodeWrongPassword, errorCode(t, app.Unlock("wrong-password-1")))
	require.NoError(t, app.Unlock("desktop-pass-1"))

	require.NoError(t, app.DeleteSecret("note", "text"))
	_, err = app.GetSecret("note", "text")
	assert.Equal(t, vault.CodeNotFound, errorCode(t, err))
}

func TestE2E_Desktop_ClipboardIsCleared(t *testing.T) {
	ui := &fakeUI{}
	app := backend.NewApp(backend.Options{Dir: t.TempDir(), UI: ui, ClipboardClearAfter: 100 * time.Millisecond})

	require.NoError(t, app.CopyToClipboard("secret-1"))
	assert.Equal(t, "secret-1", ui.clip())
	assert.Eventually(t, func() bool { return ui.clip() == "" }, time.Second, 10*time.Millisecond)

	// Если пользователь скопировал что-то своё, это не стирается.
	require.NoError(t, app.CopyToClipboard("secret-2"))
	require.NoError(t, ui.ClipboardSet("user text"))
	time.Sleep(250 * time.Millisecond)
	assert.Equal(t, "user text", ui.clip())
}

func TestE2E_Desktop_SettingsPersistAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	first := newDesktopApp(t, dir, &fakeUI{})
	require.NoError(t, first.SaveSettings(serverAddr, "", 7))
	login := "desk-" + uuid.NewString()
	require.NoError(t, first.Register(login, "desktop-pass-1"))
	_, err := first.SignIn(login, "desktop-pass-1")
	require.NoError(t, err)
	backend.Shutdown(first)

	second := newDesktopApp(t, dir, &fakeUI{})
	state := second.GetState()
	assert.Equal(t, serverAddr, state.ServerAddress)
	assert.Equal(t, 7, state.AutoLockMinutes)
	assert.Equal(t, login, state.Login, "login is remembered")
	assert.False(t, state.Unlocked, "keys are not persisted")
	require.NoError(t, second.Unlock("desktop-pass-1"))
}

func TestE2E_Desktop_TemporaryPassword(t *testing.T) {
	admin := newClientKit(t)
	tempPassword, _, err := admin.Auth.CreateTemporaryUser(context.Background(), admin.Login, domain.DefaultKDFParams)
	require.NoError(t, err)

	app := newDesktopApp(t, t.TempDir(), &fakeUI{})
	require.NoError(t, app.SaveSettings(serverAddr, "", 5))

	changeRequired, err := app.SignIn(admin.Login, tempPassword)
	require.NoError(t, err)
	require.True(t, changeRequired)
	assert.False(t, app.GetState().Unlocked)

	assert.Equal(t, backend.CodeValidation, errorCode(t, app.CompletePasswordChange(tempPassword, tempPassword)))
	require.NoError(t, app.CompletePasswordChange(tempPassword, "own-desktop-pass"))
	assert.True(t, app.GetState().Unlocked)
}
