package command_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/client/command"
	"github.com/F3dosik/GophKeeper/internal/client/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	cmds := command.New(nil, nil, &config.Config{})
	assert.NotNil(t, cmds)
}

func TestExecute_Version(t *testing.T) {
	orig := command.Version
	command.Version = "1.2.3"
	t.Cleanup(func() { command.Version = orig })

	out := captureStdout(t, func() {
		os.Args = []string{"gophkeeper", "version"}
		err := command.New(nil, nil, &config.Config{}).Execute()
		require.NoError(t, err)
	})

	assert.Contains(t, out, "Version: 1.2.3")
	assert.Contains(t, out, "Build date:")
}

func TestExecute_UnknownCommand(t *testing.T) {
	cmds := command.New(nil, nil, &config.Config{})
	// Silence Cobra's own stderr output, redirect it to buf.
	buf := &bytes.Buffer{}
	os.Args = []string{"gophkeeper", "nonexistent"}
	// We can't easily inject stderr; Execute returns the error.
	err := cmds.Execute()
	assert.Error(t, err)
	_ = buf
}

func TestLogout_RemovesSessionFile(t *testing.T) {
	sessionPath := filepath.Join(t.TempDir(), "session")
	require.NoError(t, os.WriteFile(sessionPath, []byte(`{"login":"u","token":"t"}`), 0600))

	cmds := command.New(nil, nil, &config.Config{SessionPath: sessionPath})
	out := captureStdout(t, func() {
		os.Args = []string{"gophkeeper", "auth", "logout"}
		err := cmds.Execute()
		require.NoError(t, err)
	})

	_, err := os.Stat(sessionPath)
	assert.True(t, os.IsNotExist(err), "session file should be removed")
	assert.Contains(t, out, "Сессия удалена")
}

func TestLogout_MissingSessionFile_NoError(t *testing.T) {
	cmds := command.New(nil, nil, &config.Config{
		SessionPath: filepath.Join(t.TempDir(), "missing"),
	})
	os.Args = []string{"gophkeeper", "auth", "logout"}
	err := cmds.Execute()
	assert.NoError(t, err)
}
