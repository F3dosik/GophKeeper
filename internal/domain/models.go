// Package domain содержит основные типы и интерфейсы системы.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// User представляет пользователя системы.
type User struct {
	ID           uuid.UUID
	Login        string
	PasswordHash string
	PasswordSalt string
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

// CredentialsSecret представляет секрет с парами логин/пароль.
type CredentialsSecret struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Metadata string `json:"metadata,omitempty"`
}

// TextSecret представляет секрет с произвольными текстовыми данными.
type TextSecret struct {
	Text     string `json:"text"`
	Metadata string `json:"metadata,omitempty"`
}

// BinarySecret представляет секрет с произвольными бинарными данными.
type BinarySecret struct {
	Data     []byte `json:"data"`
	Metadata string `json:"metadata,omitempty"`
}

// CardSecret представляет секрет с данными банковских карт.
type CardSecret struct {
	Number   string `json:"number"`
	Holder   string `json:"holder"`
	Expiry   string `json:"expiry"`
	CVV      string `json:"cvv"`
	Metadata string `json:"metadata,omitempty"`
}

// Credentials содержит учётные данные пользователя.
type Credentials struct {
	Login string
	// MasterKey — хэш мастер-пароля, полученный через Argon2id.
	MasterKey string
}
