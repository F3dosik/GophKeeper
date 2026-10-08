package grpcclient

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransportCredentials(t *testing.T) {
	t.Run("TLS with system roots by default", func(t *testing.T) {
		creds, err := transportCredentials("", false)
		require.NoError(t, err)
		assert.Equal(t, "tls", creds.Info().SecurityProtocol)
	})

	t.Run("insecure only when explicitly allowed", func(t *testing.T) {
		creds, err := transportCredentials("", true)
		require.NoError(t, err)
		assert.Equal(t, "insecure", creds.Info().SecurityProtocol)
	})

	t.Run("insecure together with cert is rejected", func(t *testing.T) {
		_, err := transportCredentials("/path/ca.crt", true)
		assert.ErrorIs(t, err, ErrInsecureWithCert)
	})

	t.Run("missing cert file", func(t *testing.T) {
		_, err := transportCredentials(filepath.Join(t.TempDir(), "nope.crt"), false)
		assert.Error(t, err)
	})

	t.Run("invalid cert file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.crt")
		require.NoError(t, os.WriteFile(path, []byte("not a cert"), 0600))
		_, err := transportCredentials(path, false)
		assert.Error(t, err)
	})
}
