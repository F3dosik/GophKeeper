package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
)

// Cipher определяет интерфейс для шифрования и расшифровки данных.
type Cipher interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// aesCipher реализует Cipher используя AES-256-GCM.
type aesCipher struct {
	gcm cipher.AEAD
}

// NewAESCipher создаёт новый AES-256-GCM шифр из ключа длиной 32 байта.
func NewAESCipher(key []byte) (Cipher, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("NewAESCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("NewAESCipher: %w", err)
	}

	return &aesCipher{gcm: gcm}, nil
}

// Encrypt шифрует plaintext используя AES-256-GCM.
// Возвращает nonce + ciphertext, nonce генерируется случайно для каждого вызова.
func (c *aesCipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}
	return c.gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt расшифровывает ciphertext используя AES-256-GCM.
// Ожидает формат nonce + ciphertext полученный из Encrypt.
func (c *aesCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < c.gcm.NonceSize() {
		return nil, fmt.Errorf("decrypt: ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:c.gcm.NonceSize()], ciphertext[c.gcm.NonceSize():]
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}
