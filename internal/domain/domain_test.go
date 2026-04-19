package domain_test

import (
	"testing"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestParseSecretType(t *testing.T) {
	tests := []struct {
		in   string
		want domain.SecretType
	}{
		{"credentials", domain.SecretTypeCredentials},
		{"text", domain.SecretTypeText},
		{"binary", domain.SecretTypeBinary},
		{"card", domain.SecretTypeCard},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := domain.ParseSecretType(tt.in)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseSecretType_Unknown(t *testing.T) {
	tests := []string{"", "unknown", "Credentials", "CARD"}
	for _, s := range tests {
		t.Run(s, func(t *testing.T) {
			_, err := domain.ParseSecretType(s)
			assert.ErrorIs(t, err, domain.ErrUnknownSecretType)
		})
	}
}

func TestCardSecret_Validate(t *testing.T) {
	valid := domain.CardSecret{
		Number: "4242 4242 4242 4242",
		Holder: "IVAN IVANOV",
		Expiry: "12/30",
		CVV:    "123",
	}

	t.Run("valid with spaces", func(t *testing.T) {
		c := valid
		assert.NoError(t, c.Validate())
	})

	t.Run("valid 4-digit CVV", func(t *testing.T) {
		c := valid
		c.CVV = "1234"
		assert.NoError(t, c.Validate())
	})

	t.Run("invalid Luhn", func(t *testing.T) {
		c := valid
		c.Number = "4242424242424243"
		assert.ErrorIs(t, c.Validate(), domain.ErrInvalidCardNumber)
	})

	t.Run("number too short", func(t *testing.T) {
		c := valid
		c.Number = "424242424242"
		assert.ErrorIs(t, c.Validate(), domain.ErrInvalidCardNumber)
	})

	t.Run("number too long", func(t *testing.T) {
		c := valid
		c.Number = "42424242424242424242"
		assert.ErrorIs(t, c.Validate(), domain.ErrInvalidCardNumber)
	})

	t.Run("non-digit in number", func(t *testing.T) {
		c := valid
		c.Number = "4242abcd42424242"
		assert.ErrorIs(t, c.Validate(), domain.ErrInvalidCardNumber)
	})

	t.Run("invalid expiry format", func(t *testing.T) {
		cases := []string{"13/25", "00/25", "1/25", "12/2025", "12-25", ""}
		for _, e := range cases {
			c := valid
			c.Expiry = e
			assert.ErrorIs(t, c.Validate(), domain.ErrInvalidCardExpiry, "expiry=%q", e)
		}
	})

	t.Run("invalid CVV", func(t *testing.T) {
		cases := []string{"12", "12345", "abc", ""}
		for _, cvv := range cases {
			c := valid
			c.CVV = cvv
			assert.ErrorIs(t, c.Validate(), domain.ErrInvalidCardCVV, "cvv=%q", cvv)
		}
	})

	t.Run("empty holder", func(t *testing.T) {
		c := valid
		c.Holder = "   "
		assert.ErrorIs(t, c.Validate(), domain.ErrInvalidCardHolder)
	})
}
