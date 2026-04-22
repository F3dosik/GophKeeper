package crypto_test

import (
	"crypto/rand"
	"testing"

	"github.com/F3dosik/GophKeeper/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAESCipher(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	t.Run("encrypt and decrypt", func(t *testing.T) {
		cipher, err := crypto.NewAESCipher(key)
		require.NoError(t, err)

		plaintext := []byte("secret data")
		ciphertext, err := cipher.Encrypt(plaintext)
		require.NoError(t, err)
		assert.NotEqual(t, plaintext, ciphertext)

		decrypted, err := cipher.Decrypt(ciphertext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("encrypt returns unique ciphertext", func(t *testing.T) {
		cipher, err := crypto.NewAESCipher(key)
		require.NoError(t, err)

		plaintext := []byte("secret data")
		ct1, err := cipher.Encrypt(plaintext)
		require.NoError(t, err)
		ct2, err := cipher.Encrypt(plaintext)
		require.NoError(t, err)
		assert.NotEqual(t, ct1, ct2)
	})

	t.Run("decrypt with short ciphertext returns error", func(t *testing.T) {
		cipher, err := crypto.NewAESCipher(key)
		require.NoError(t, err)

		_, err = cipher.Decrypt([]byte("short"))
		assert.Error(t, err)
	})

	t.Run("decrypt with tampered ciphertext returns error", func(t *testing.T) {
		cipher, err := crypto.NewAESCipher(key)
		require.NoError(t, err)

		ciphertext, err := cipher.Encrypt([]byte("secret"))
		require.NoError(t, err)
		ciphertext[len(ciphertext)-1] ^= 0xff

		_, err = cipher.Decrypt(ciphertext)
		assert.Error(t, err)
	})

	t.Run("invalid key length returns error", func(t *testing.T) {
		_, err := crypto.NewAESCipher([]byte("short"))
		assert.Error(t, err)
	})
}
