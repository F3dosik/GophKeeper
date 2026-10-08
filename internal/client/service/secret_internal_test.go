package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPadPayload(t *testing.T) {
	for _, size := range []int{0, 1, 100, 180, 190, 200, 255, 256, 1000, 5000} {
		payload := &domain.SecretPayload{
			Name: "name",
			Type: domain.SecretTypeText,
			Data: json.RawMessage(`{"text":"` + strings.Repeat("x", size) + `"}`),
		}

		padded, err := padPayload(payload)
		require.NoError(t, err)
		assert.Zero(t, len(padded)%paddingBlock, "size %d: length %d is not a multiple of %d", size, len(padded), paddingBlock)

		var got domain.SecretPayload
		require.NoError(t, json.Unmarshal(padded, &got))
		assert.Equal(t, *payload, got, "size %d: padding must not change the payload", size)
	}
}

func TestPadPayload_HidesLength(t *testing.T) {
	short, err := padPayload(&domain.SecretPayload{Name: "a", Type: domain.SecretTypeCredentials,
		Data: json.RawMessage(`{"login":"u","password":"123456"}`)})
	require.NoError(t, err)
	long, err := padPayload(&domain.SecretPayload{Name: "a", Type: domain.SecretTypeCredentials,
		Data: json.RawMessage(`{"login":"u","password":"correct-horse-battery-staple-42"}`)})
	require.NoError(t, err)

	assert.Equal(t, len(short), len(long))
}
