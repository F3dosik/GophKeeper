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
	_, _ = mac.Write([]byte(name + ":" + string(secretType)))
	return hex.EncodeToString(mac.Sum(nil))
}

// GenerateSaltByLogin детерминированно выводит 16-байтную соль из login и serverKey
// через HMAC-SHA256. Используется сервером для ответа на GetSalt о несуществующем
// пользователе: стабильность по логину делает фиктивную соль неотличимой от реальной
// для атакующего, не знающего serverKey, и блокирует перечисление пользователей.
func GenerateSaltByLogin(login string, serverKey []byte) []byte {
	mac := hmac.New(sha256.New, serverKey)
	_, _ = mac.Write([]byte(login))
	return mac.Sum(nil)[:16]
}
