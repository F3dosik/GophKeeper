package command_test

import (
	"bytes"
	"github.com/F3dosik/GophKeeper/internal/domain"
	"os"
	"path/filepath"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/client/command"
	"github.com/F3dosik/GophKeeper/internal/client/config"
	"github.com/F3dosik/GophKeeper/internal/client/mocks"
	"github.com/F3dosik/GophKeeper/internal/client/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	cmds := command.New(nil, nil, nil, &config.Config{})
	assert.NotNil(t, cmds)
}

func TestExecute_Version(t *testing.T) {
	orig := command.Version
	command.Version = "1.2.3"
	t.Cleanup(func() { command.Version = orig })

	out := captureStdout(t, func() {
		os.Args = []string{"gophkeeper", "version"}
		err := command.New(nil, nil, nil, &config.Config{}).Execute()
		require.NoError(t, err)
	})

	assert.Contains(t, out, "Version: 1.2.3")
	assert.Contains(t, out, "Build date:")
}

func TestExecute_UnknownCommand(t *testing.T) {
	cmds := command.New(nil, nil, nil, &config.Config{})
	// Silence Cobra's own stderr output, redirect it to buf.
	buf := &bytes.Buffer{}
	os.Args = []string{"gophkeeper", "nonexistent"}
	// We can't easily inject stderr; Execute returns the error.
	err := cmds.Execute()
	assert.Error(t, err)
	_ = buf
}

// logoutCommands возвращает Commands с реальным AuthService поверх мока gRPC-клиента,
// который ожидает Logout с флагом allSessions.
func logoutCommands(t *testing.T, sessionPath string, allSessions bool) *command.Commands {
	t.Helper()

	mockAuth := mocks.NewAuthClient(t)
	mockAuth.On("Logout", mock.Anything, allSessions).Return(nil)
	authSvc := service.NewAuthService(mockAuth, sessionPath, nil)
	return command.New(authSvc, nil, nil, &config.Config{SessionPath: sessionPath})
}

func TestLogout_RevokesTokenAndRemovesSessionFile(t *testing.T) {
	sessionPath := filepath.Join(t.TempDir(), "session")
	require.NoError(t, os.WriteFile(sessionPath, []byte(`{"login":"u","token":"t"}`), 0600))

	cmds := logoutCommands(t, sessionPath, false)
	out := captureStdout(t, func() {
		os.Args = []string{"gophkeeper", "auth", "logout"}
		err := cmds.Execute()
		require.NoError(t, err)
	})

	_, err := os.Stat(sessionPath)
	assert.True(t, os.IsNotExist(err), "session file should be removed")
	assert.Contains(t, out, "токен отозван")
}

func TestLogout_AllSessions(t *testing.T) {
	sessionPath := filepath.Join(t.TempDir(), "session")
	require.NoError(t, os.WriteFile(sessionPath, []byte(`{"login":"u","token":"t"}`), 0600))

	cmds := logoutCommands(t, sessionPath, true)
	out := captureStdout(t, func() {
		os.Args = []string{"gophkeeper", "auth", "logout", "--all"}
		err := cmds.Execute()
		require.NoError(t, err)
	})

	assert.Contains(t, out, "на всех устройствах")
}

func TestLogout_MissingSessionFile_NoError(t *testing.T) {
	cmds := command.New(nil, nil, nil, &config.Config{
		SessionPath: filepath.Join(t.TempDir(), "missing"),
	})
	os.Args = []string{"gophkeeper", "auth", "logout"}
	err := cmds.Execute()
	assert.NoError(t, err)
}

// withStdin подменяет стандартный ввод на input на время fn.
func withStdin(t *testing.T, input string, fn func()) {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, err = w.WriteString(input)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig })
	fn()
}

func TestAdminDeleteUser(t *testing.T) {
	run := func(admin *mocks.AdminClient, args ...string) error {
		os.Args = append([]string{"gophkeeper", "admin", "delete-user"}, args...)
		return command.New(nil, nil, admin, &config.Config{}).Execute()
	}

	t.Run("deletes with --yes", func(t *testing.T) {
		admin := mocks.NewAdminClient(t)
		admin.On("DeleteUser", mock.Anything, "bob").Return(nil)
		out := captureStdout(t, func() { require.NoError(t, run(admin, "bob", "--yes")) })
		assert.Contains(t, out, "bob удалён")
	})

	t.Run("confirmation by typing the login", func(t *testing.T) {
		admin := mocks.NewAdminClient(t)
		admin.On("DeleteUser", mock.Anything, "bob").Return(nil)
		withStdin(t, "bob\n", func() {
			captureStdout(t, func() { require.NoError(t, run(admin, "bob")) })
		})
	})

	t.Run("wrong confirmation deletes nothing", func(t *testing.T) {
		admin := mocks.NewAdminClient(t) // без ожиданий: вызов DeleteUser провалит тест
		withStdin(t, "alice\n", func() {
			captureStdout(t, func() { assert.Error(t, run(admin, "bob")) })
		})
	})

	t.Run("public port explains how to reach the admin port", func(t *testing.T) {
		admin := mocks.NewAdminClient(t)
		admin.On("DeleteUser", mock.Anything, "bob").Return(domain.ErrNotSupported)
		var err error
		captureStdout(t, func() { err = run(admin, "bob", "--yes") })
		assert.ErrorContains(t, err, "ADMIN_PORT")
	})
}
