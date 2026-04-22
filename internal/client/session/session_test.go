package session_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/client/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session")
	want := &session.Session{Login: "user", Token: "jwt-token"}

	err := session.Save(path, want)
	require.NoError(t, err)

	got, err := session.Load(path)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestSave_CreatesParentDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "deep", "session")

	err := session.Save(path, &session.Session{Login: "user", Token: "t"})
	require.NoError(t, err)

	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestSave_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permissions not applicable on Windows")
	}
	path := filepath.Join(t.TempDir(), "session")

	err := session.Save(path, &session.Session{Login: "user", Token: "t"})
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestLoad_FileNotExist(t *testing.T) {
	_, err := session.Load(filepath.Join(t.TempDir(), "missing"))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestLoad_InvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session")
	require.NoError(t, os.WriteFile(path, []byte("not-json"), 0600))

	_, err := session.Load(path)
	assert.Error(t, err)
}

func TestSave_WriteError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permissions not applicable on Windows")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) })

	err := session.Save(filepath.Join(dir, "nested", "session"), &session.Session{Login: "u", Token: "t"})
	assert.Error(t, err)
}
