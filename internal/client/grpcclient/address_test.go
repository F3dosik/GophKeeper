package grpcclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeAddress(t *testing.T) {
	valid := map[string]string{
		"192.168.1.5":          "192.168.1.5:50051",
		" 192.168.1.5 ":        "192.168.1.5:50051",
		"192.168.1.5:6000":     "192.168.1.5:6000",
		"localhost":            "localhost:50051",
		"keeper.example.com":   "keeper.example.com:50051",
		"keeper.example.com:1": "keeper.example.com:1",
		"::1":                  "[::1]:50051",
		"fe80::1":              "[fe80::1]:50051",
		"[fe80::1]":            "[fe80::1]:50051",
		"[fe80::1]:6000":       "[fe80::1]:6000",
	}
	for in, want := range valid {
		got, err := NormalizeAddress(in)
		if assert.NoError(t, err, in) {
			assert.Equal(t, want, got, in)
		}
	}

	for _, in := range []string{"", "  ", "host:", ":50051", "a:b:c", "http://host", "my host", "host:0", "host:70000", "host:port"} {
		_, err := NormalizeAddress(in)
		assert.ErrorIs(t, err, ErrBadAddress, in)
	}
}
