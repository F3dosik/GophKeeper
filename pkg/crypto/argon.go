package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

// Параметры Argon2id согласно рекомендациям OWASP.
const (
	timeCost = 1
	memory   = 64 * 1024
	threads  = 4
	keyLen   = 32
)

// Пустая соль для явной передачи в hkdf.
var noSalt []byte

const (
	// InfoEncryption используется для деривации ключа шифрования AES-256-GCM.
	InfoEncryption = "encryption"
	// InfoBlindIndex используется для деривации ключа HMAC-SHA256 blind index.
	InfoBlindIndex = "blind-index"
)

// DeriveKey возвращает ключ из пароля и соли используя Argon2id.
func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, timeCost, memory, threads, keyLen)
}

// GenerateSalt генерирует случайную соль длиной 16 байт.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	return salt, nil
}

// HKDF выводит ключ длиной 32 байта из masterKey используя HKDF-SHA256.
// info задаёт контекст деривации и гарантирует независимость ключей.
func HKDF(materKey []byte, info string) ([]byte, error) {
	reader := hkdf.New(sha256.New, materKey, noSalt, []byte(info))
	key := make([]byte, 32)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, fmt.Errorf("hkdf exapnd: %w", err)
	}
	return key, nil
}
