package kdfpin_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/client/kdfpin"
	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var params = domain.KDFParams{Time: 3, MemoryKiB: 64 * 1024, Threads: 4}

func TestStore_RememberGetForget(t *testing.T) {
	path := filepath.Join(t.TempDir(), kdfpin.FileName)
	s := kdfpin.New(path)

	_, ok, err := s.Get("srv:50051", "alice")
	require.NoError(t, err)
	assert.False(t, ok, "файла ещё нет")

	require.NoError(t, s.Remember("srv:50051", "alice", params))

	got, ok, err := kdfpin.New(path).Get("srv:50051", "Alice")
	require.NoError(t, err)
	require.True(t, ok, "логины регистронезависимы, параметры читаются из файла")
	assert.Equal(t, params, got)

	_, ok, err = s.Get("other:50051", "alice")
	require.NoError(t, err)
	assert.False(t, ok, "параметры привязаны к серверу")

	require.NoError(t, s.Forget("srv:50051", "ALICE"))
	_, ok, err = s.Get("srv:50051", "alice")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestStore_FilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", kdfpin.FileName)
	require.NoError(t, kdfpin.New(path).Remember("srv", "alice", params))

	info, err := os.Stat(path)
	require.NoError(t, err)
	if os.PathSeparator == '/' {
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

// Повреждённый файл — ошибка: иначе испорченный файл молча отключил бы защиту.
func TestStore_CorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), kdfpin.FileName)
	require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o600))

	_, _, err := kdfpin.New(path).Get("srv", "alice")
	assert.ErrorContains(t, err, "повреждён")
}

func TestStore_Nil(t *testing.T) {
	var s *kdfpin.Store
	require.NoError(t, s.Remember("srv", "alice", params))
	_, ok, err := s.Get("srv", "alice")
	require.NoError(t, err)
	assert.False(t, ok)
	require.NoError(t, s.Forget("srv", "alice"))

	assert.Nil(t, kdfpin.NextTo(""))
}
