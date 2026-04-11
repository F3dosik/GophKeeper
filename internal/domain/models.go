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
