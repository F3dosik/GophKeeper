package service

import (
	"strings"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestValidateLogin(t *testing.T) {
	valid := []string{"alice", "Алиса", "user.name+tag@example.com", strings.Repeat("я", maxLoginLength)}
	for _, login := range valid {
		assert.NoError(t, validateLogin(login), login)
	}

	invalid := map[string]string{
		"empty":         "",
		"too long":      strings.Repeat("a", maxLoginLength+1),
		"leading space": " alice",
		"trailing tab":  "alice\t",
		"newline":       "ali\nce",
		"null byte":     "ali\x00ce",
		"invalid utf8":  "ali\xffce",
	}
	for name, login := range invalid {
		assert.ErrorIs(t, validateLogin(login), domain.ErrInvalidArgument, name)
	}
}

func TestValidateKeys(t *testing.T) {
	assert.NoError(t, validateAuthKey(make([]byte, 32)))
	assert.ErrorIs(t, validateAuthKey(nil), domain.ErrInvalidArgument)
	assert.ErrorIs(t, validateAuthKey(make([]byte, 31)), domain.ErrInvalidArgument)

	assert.NoError(t, validateSalt(make([]byte, 16)))
	assert.ErrorIs(t, validateSalt(make([]byte, 1<<20)), domain.ErrInvalidArgument)
}

func TestValidateBlindIndex(t *testing.T) {
	assert.NoError(t, validateBlindIndex(strings.Repeat("0a", 32)))
	assert.ErrorIs(t, validateBlindIndex(""), domain.ErrInvalidArgument)
	assert.ErrorIs(t, validateBlindIndex(strings.Repeat("A", 64)), domain.ErrInvalidArgument, "uppercase")
	assert.ErrorIs(t, validateBlindIndex(strings.Repeat("g", 64)), domain.ErrInvalidArgument, "non-hex")
	assert.ErrorIs(t, validateBlindIndex(strings.Repeat("a", 65)), domain.ErrInvalidArgument, "too long")
}
