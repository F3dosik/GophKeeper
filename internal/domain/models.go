// Package domain содержит основные типы и интерфейсы системы.
package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// User представляет пользователя системы.
type User struct {
	ID           uuid.UUID
	Login        string
	PasswordHash []byte
	PasswordSalt []byte
	CreatedAt    time.Time
}

// Secret представляет зашифрованный секрет пользователя.
type Secret struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	BlindIndex string
	Data       []byte
	UpdatedAt  time.Time
	CreatedAt  time.Time
}

// SecretType определяет тип хранимого секрета.
type SecretType string

const (
	SecretTypeCredentials SecretType = "credentials" // пары логин/пароль
	SecretTypeText        SecretType = "text"        // произвольный текст
	SecretTypeBinary      SecretType = "binary"      // произвольные бинарные данные
	SecretTypeCard        SecretType = "card"        // данные банковской карты
)

// ErrUnknownSecretType возвращается при попытке использовать неизвестный тип секрета.
var ErrUnknownSecretType = errors.New("unknown secret type")

// ParseSecretType валидирует строку и возвращает SecretType.
// Возвращает ErrUnknownSecretType, если значение не соответствует ни одному типу.
func ParseSecretType(s string) (SecretType, error) {
	switch SecretType(s) {
	case SecretTypeCredentials, SecretTypeText, SecretTypeBinary, SecretTypeCard:
		return SecretType(s), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownSecretType, s)
	}
}

// CredentialsSecret представляет секрет с парами логин/пароль.
type CredentialsSecret struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// TextSecret представляет секрет с произвольными текстовыми данными.
type TextSecret struct {
	Text string `json:"text"`
}

// BinarySecret представляет секрет с произвольными бинарными данными.
type BinarySecret struct {
	Data []byte `json:"data"`
}

// CardSecret представляет секрет с данными банковских карт.
type CardSecret struct {
	Number string `json:"number"`
	Holder string `json:"holder"`
	Expiry string `json:"expiry"`
	CVV    string `json:"cvv"`
}

// SecretPayload представляет секрет для шифрования.
type SecretPayload struct {
	Name     string          `json:"name"`
	Type     SecretType      `json:"type"`
	Data     json.RawMessage `json:"data"`
	Metadata string          `json:"metadata,omitempty"`
}

// SecretInfo представляет секрет для отображения пользователю.
type SecretInfo struct {
	SecretPayload
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Credentials содержит учётные данные пользователя.
type Credentials struct {
	Login string
	// MasterKey — хэш мастер-пароля, полученный через Argon2id.
	MasterKey []byte
}
