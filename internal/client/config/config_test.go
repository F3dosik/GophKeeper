package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/F3dosik/Shron/internal/client/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolate подменяет домашний каталог временным и убирает переменные клиента:
// Load переносит ~/.gophkeeper, и тест не должен трогать настоящий каталог.
func isolate(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, name := range []string{"SERVER", "SESSION", "TLS_CERT", "INSECURE"} {
		for _, prefix := range []string{"SHRON_", "GOPHKEEPER_"} {
			t.Setenv(prefix+name, "")
			require.NoError(t, os.Unsetenv(prefix+name))
		}
	}
	return home
}

func TestLoad_Defaults(t *testing.T) {
	isolate(t)
	t.Setenv("SHRON_SERVER", "")
	t.Setenv("SHRON_SESSION", "")
	t.Setenv("SHRON_TLS_CERT", "")
	t.Setenv("SHRON_INSECURE", "")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.False(t, cfg.Insecure, "TLS must be enabled by default")
	assert.Equal(t, "localhost:50051", cfg.ServerAddress)
	assert.NotEmpty(t, cfg.SessionPath)
	assert.False(t, strings.HasPrefix(cfg.SessionPath, "~/"), "SessionPath should be expanded")
	assert.Empty(t, cfg.TLSCertPath)
}

func TestLoad_FromEnv(t *testing.T) {
	isolate(t)
	t.Setenv("SHRON_SERVER", "example.com:443")
	t.Setenv("SHRON_SESSION", "/tmp/custom-session")
	t.Setenv("SHRON_TLS_CERT", "/etc/certs/ca.pem")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "example.com:443", cfg.ServerAddress)
	assert.Equal(t, "/tmp/custom-session", cfg.SessionPath)
	assert.Equal(t, "/etc/certs/ca.pem", cfg.TLSCertPath)
}

func TestLoad_ExpandsHomeInSessionPath(t *testing.T) {
	isolate(t)
	t.Setenv("SHRON_SESSION", "~/mysession")

	cfg, err := config.Load()

	require.NoError(t, err)
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "mysession"), cfg.SessionPath)
}

func TestLoad_DefaultSessionPathIsExpanded(t *testing.T) {
	isolate(t)
	t.Setenv("SHRON_SESSION", "")

	cfg, err := config.Load()

	require.NoError(t, err)
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(cfg.SessionPath, home),
		"default session path should start with user home, got %q", cfg.SessionPath)
}

func TestLoad_PathWithoutTildeUnchanged(t *testing.T) {
	isolate(t)
	t.Setenv("SHRON_SESSION", "/absolute/path")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "/absolute/path", cfg.SessionPath)
}

func TestLoad_Insecure(t *testing.T) {
	isolate(t)
	t.Setenv("SHRON_INSECURE", "true")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.True(t, cfg.Insecure)
}

func TestLoad_ServerPortIsOptional(t *testing.T) {
	isolate(t)
	t.Setenv("SHRON_SERVER", "192.168.1.5")
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "192.168.1.5:50051", cfg.ServerAddress)

	t.Setenv("SHRON_SERVER", "http://bad")
	_, err = config.Load()
	assert.Error(t, err)
}

func TestLoad_LegacyEnv(t *testing.T) {
	isolate(t)
	t.Setenv("GOPHKEEPER_SERVER", "old.example.com")
	t.Setenv("GOPHKEEPER_INSECURE", "true")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "old.example.com:50051", cfg.ServerAddress, "names before the rename still work")
	assert.True(t, cfg.Insecure)

	t.Setenv("SHRON_SERVER", "new.example.com")
	cfg, err = config.Load()
	require.NoError(t, err)
	assert.Equal(t, "new.example.com:50051", cfg.ServerAddress, "the new name wins")
	assert.Empty(t, os.Getenv("SHRON_INSECURE"), "Load does not modify the process environment")

	assert.Equal(t, "new.example.com", config.Getenv("SERVER"))
	assert.Equal(t, "true", config.Getenv("INSECURE"))
}

func TestLoad_MigratesLegacyDataDir(t *testing.T) {
	home := isolate(t)
	legacy := filepath.Join(home, ".gophkeeper")
	require.NoError(t, os.MkdirAll(legacy, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "session"), []byte("old"), 0o600))

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".shron", "session"), cfg.SessionPath)
	data, err := os.ReadFile(cfg.SessionPath)
	require.NoError(t, err)
	assert.Equal(t, "old", string(data), "the old session is moved")
	assert.NoDirExists(t, legacy)
}

func TestMigrateDir(t *testing.T) {
	dir := t.TempDir()
	oldDir, newDir := filepath.Join(dir, "old"), filepath.Join(dir, "nested", "new")

	require.NoError(t, config.MigrateDir(oldDir, newDir), "nothing to migrate")
	assert.NoDirExists(t, newDir)

	require.NoError(t, os.MkdirAll(oldDir, 0o700))
	require.NoError(t, os.MkdirAll(newDir, 0o700))
	require.NoError(t, config.MigrateDir(oldDir, newDir))
	assert.DirExists(t, oldDir, "an existing new directory is never overwritten")
}
