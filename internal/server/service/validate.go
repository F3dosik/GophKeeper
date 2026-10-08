package service

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/F3dosik/GophKeeper/internal/domain"
)

// Ограничения на входные данные. Клиент формирует ключи и blind index фиксированной
// длины, поэтому всё остальное — ошибка клиента или попытка нагрузить сервер.
const (
	maxLoginLength = 64
	authKeyLength  = 32
	saltLength     = 16
	// blindIndexLength — длина hex-кодированного HMAC-SHA256.
	blindIndexLength = 64
)

// validateLogin проверяет логин: 1–64 символа UTF-8 без управляющих символов
// и без пробелов по краям.
func validateLogin(login string) error {
	if !utf8.ValidString(login) {
		return fmt.Errorf("%w: login must be valid UTF-8", domain.ErrInvalidArgument)
	}
	if n := utf8.RuneCountInString(login); n == 0 || n > maxLoginLength {
		return fmt.Errorf("%w: login must be 1-%d characters", domain.ErrInvalidArgument, maxLoginLength)
	}
	if strings.TrimSpace(login) != login {
		return fmt.Errorf("%w: login must not start or end with whitespace", domain.ErrInvalidArgument)
	}
	for _, r := range login {
		if unicode.IsControl(r) {
			return fmt.Errorf("%w: login must not contain control characters", domain.ErrInvalidArgument)
		}
	}
	return nil
}

// validateAuthKey проверяет длину ключа аутентификации.
func validateAuthKey(authKey []byte) error {
	if len(authKey) != authKeyLength {
		return fmt.Errorf("%w: auth key must be %d bytes", domain.ErrInvalidArgument, authKeyLength)
	}
	return nil
}

// validateSalt проверяет длину соли.
func validateSalt(salt []byte) error {
	if len(salt) != saltLength {
		return fmt.Errorf("%w: salt must be %d bytes", domain.ErrInvalidArgument, saltLength)
	}
	return nil
}

// validateBlindIndex проверяет, что blind index — 64 символа в нижнем регистре hex.
func validateBlindIndex(blindIndex string) error {
	if len(blindIndex) != blindIndexLength {
		return fmt.Errorf("%w: blind index must be %d hex characters", domain.ErrInvalidArgument, blindIndexLength)
	}
	for _, c := range blindIndex {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return fmt.Errorf("%w: blind index must be lowercase hex", domain.ErrInvalidArgument)
		}
	}
	return nil
}
