package crypto_test

import (
	"encoding/hex"
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

func TestAuthKeyIndependence(t *testing.T) {
	master := []byte("masterkey")
	authKey, err := crypto.HKDF(master, crypto.InfoAuth)
	require.NoError(t, err)
	encKey, err := crypto.HKDF(master, crypto.InfoEncryption)
	require.NoError(t, err)
	hmacKey, err := crypto.HKDF(master, crypto.InfoBlindIndex)
	require.NoError(t, err)

	assert.NotEqual(t, authKey, encKey)
	assert.NotEqual(t, authKey, hmacKey)
	assert.NotEqual(t, authKey, master)
}

func TestHashAuthKey(t *testing.T) {
	t.Run("returns 32 bytes", func(t *testing.T) {
		assert.Len(t, crypto.HashAuthKey([]byte("key")), 32)
	})

	t.Run("deterministic and differs from input", func(t *testing.T) {
		key := make([]byte, 32)
		h1 := crypto.HashAuthKey(key)
		h2 := crypto.HashAuthKey(key)
		assert.Equal(t, h1, h2)
		assert.NotEqual(t, key, h1)
	})
}

// Вектор совпадает с тем, что вычисляет SQL в migrations/000002_auth_key_hash.up.sql:
// изменение деривации authKey сломает аутентификацию мигрированных пользователей.
func TestHashAuthKey_MatchesMigrationVector(t *testing.T) {
	master := make([]byte, 32)
	for i := range master {
		master[i] = byte(i)
	}
	authKey, err := crypto.HKDF(master, crypto.InfoAuth)
	require.NoError(t, err)
	assert.Equal(t, "4d3f61ca7586d06c91f93768457312a3c03e9de00767956966fd379a73813ee5", hex.EncodeToString(crypto.HashAuthKey(authKey)))
}
