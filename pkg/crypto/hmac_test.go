package crypto_test

import (
	"crypto/rand"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/pkg/crypto"
	"github.com/stretchr/testify/assert"
)

func TestBlindIndex(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	t.Run("same input returns same index", func(t *testing.T) {
		idx1 := crypto.BlindIndex("github", domain.SecretTypeCredentials, key)
		idx2 := crypto.BlindIndex("github", domain.SecretTypeCredentials, key)
		assert.Equal(t, idx1, idx2)
	})

	t.Run("different name returns different index", func(t *testing.T) {
		idx1 := crypto.BlindIndex("github", domain.SecretTypeCredentials, key)
		idx2 := crypto.BlindIndex("gitlab", domain.SecretTypeCredentials, key)
		assert.NotEqual(t, idx1, idx2)
	})

	t.Run("different type returns different index", func(t *testing.T) {
		idx1 := crypto.BlindIndex("github", domain.SecretTypeCredentials, key)
		idx2 := crypto.BlindIndex("github", domain.SecretTypeText, key)
		assert.NotEqual(t, idx1, idx2)
	})

	t.Run("different key returns different index", func(t *testing.T) {
		key2 := make([]byte, 32)
		rand.Read(key2)
		idx1 := crypto.BlindIndex("github", domain.SecretTypeCredentials, key)
		idx2 := crypto.BlindIndex("github", domain.SecretTypeCredentials, key2)
		assert.NotEqual(t, idx1, idx2)
	})
}
