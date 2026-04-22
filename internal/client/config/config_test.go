package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/client/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("GOPHKEEPER_SERVER", "")
	t.Setenv("GOPHKEEPER_SESSION", "")
	t.Setenv("GOPHKEEPER_TLS_CERT", "")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "localhost:50051", cfg.ServerAddress)
	assert.NotEmpty(t, cfg.SessionPath)
	assert.False(t, strings.HasPrefix(cfg.SessionPath, "~/"), "SessionPath should be expanded")
	assert.Empty(t, cfg.TLSCertPath)
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("GOPHKEEPER_SERVER", "example.com:443")
	t.Setenv("GOPHKEEPER_SESSION", "/tmp/custom-session")
	t.Setenv("GOPHKEEPER_TLS_CERT", "/etc/certs/ca.pem")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "example.com:443", cfg.ServerAddress)
	assert.Equal(t, "/tmp/custom-session", cfg.SessionPath)
	assert.Equal(t, "/etc/certs/ca.pem", cfg.TLSCertPath)
}

func TestLoad_ExpandsHomeInSessionPath(t *testing.T) {
	t.Setenv("GOPHKEEPER_SESSION", "~/mysession")

	cfg, err := config.Load()

	require.NoError(t, err)
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "mysession"), cfg.SessionPath)
}

func TestLoad_DefaultSessionPathIsExpanded(t *testing.T) {
	t.Setenv("GOPHKEEPER_SESSION", "")

	cfg, err := config.Load()

	require.NoError(t, err)
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(cfg.SessionPath, home),
		"default session path should start with user home, got %q", cfg.SessionPath)
}

func TestLoad_PathWithoutTildeUnchanged(t *testing.T) {
	t.Setenv("GOPHKEEPER_SESSION", "/absolute/path")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "/absolute/path", cfg.SessionPath)
}
