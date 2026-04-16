package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/F3dosik/GophKeeper/internal/domain"
)

// BlindIndex вычисляет HMAC-SHA256 от name и secretType используя key.
// Возвращает hex-encoded строку для использования как blind index.
func BlindIndex(name string, secretType domain.SecretType, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(name + ":" + string(secretType)))
	return hex.EncodeToString(mac.Sum(nil))
}
