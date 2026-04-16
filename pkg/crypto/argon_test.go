package crypto_test

import (
	"testing"

	"github.com/F3dosik/GophKeeper/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSalt(t *testing.T) {
	t.Run("returns 16 bytes", func(t *testing.T) {
		salt, err := crypto.GenerateSalt()
		require.NoError(t, err)
		assert.Len(t, salt, 16)
	})

	t.Run("returns unique salts", func(t *testing.T) {
		salt1, err := crypto.GenerateSalt()
		require.NoError(t, err)
		salt2, err := crypto.GenerateSalt()
		require.NoError(t, err)
		assert.NotEqual(t, salt1, salt2)
	})
}

func TestDeriveKey(t *testing.T) {
	t.Run("returns 32 bytes", func(t *testing.T) {
		salt, _ := crypto.GenerateSalt()
		key := crypto.DeriveKey("password", salt)
		assert.Len(t, key, 32)
	})

	t.Run("same input returns same key", func(t *testing.T) {
		salt, _ := crypto.GenerateSalt()
		key1 := crypto.DeriveKey("password", salt)
		key2 := crypto.DeriveKey("password", salt)
		assert.Equal(t, key1, key2)
	})

	t.Run("different password returns different key", func(t *testing.T) {
		salt, _ := crypto.GenerateSalt()
		key1 := crypto.DeriveKey("password1", salt)
		key2 := crypto.DeriveKey("password2", salt)
		assert.NotEqual(t, key1, key2)
	})

	t.Run("different salt returns different key", func(t *testing.T) {
		salt1, _ := crypto.GenerateSalt()
		salt2, _ := crypto.GenerateSalt()
		key1 := crypto.DeriveKey("password", salt1)
		key2 := crypto.DeriveKey("password", salt2)
		assert.NotEqual(t, key1, key2)
	})
}

func TestHKDF(t *testing.T) {
	t.Run("returns 32 bytes", func(t *testing.T) {
		key, err := crypto.HKDF([]byte("masterkey"), crypto.InfoEncryption)
		require.NoError(t, err)
		assert.Len(t, key, 32)
	})

	t.Run("same input returns same key", func(t *testing.T) {
		key1, err := crypto.HKDF([]byte("masterkey"), crypto.InfoEncryption)
		require.NoError(t, err)
		key2, err := crypto.HKDF([]byte("masterkey"), crypto.InfoEncryption)
		require.NoError(t, err)
		assert.Equal(t, key1, key2)
	})

	t.Run("different info returns different keys", func(t *testing.T) {
		key1, err := crypto.HKDF([]byte("masterkey"), crypto.InfoEncryption)
		require.NoError(t, err)
		key2, err := crypto.HKDF([]byte("masterkey"), crypto.InfoBlindIndex)
		require.NoError(t, err)
		assert.NotEqual(t, key1, key2)
	})
}
