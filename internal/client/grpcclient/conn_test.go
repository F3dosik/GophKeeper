package grpcclient

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCAPEM возвращает самоподписанный CA-сертификат в PEM.
func testCAPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestTransportCredentials(t *testing.T) {
	t.Run("TLS with system roots by default", func(t *testing.T) {
		creds, err := transportCredentials(nil, false)
		require.NoError(t, err)
		assert.Equal(t, "tls", creds.Info().SecurityProtocol)
	})

	t.Run("TLS with CA from PEM bytes", func(t *testing.T) {
		creds, err := transportCredentials(testCAPEM(t), false)
		require.NoError(t, err)
		assert.Equal(t, "tls", creds.Info().SecurityProtocol)
	})

	t.Run("insecure only when explicitly allowed", func(t *testing.T) {
		creds, err := transportCredentials(nil, true)
		require.NoError(t, err)
		assert.Equal(t, "insecure", creds.Info().SecurityProtocol)
	})

	t.Run("insecure together with cert is rejected", func(t *testing.T) {
		_, err := transportCredentials(testCAPEM(t), true)
		assert.ErrorIs(t, err, ErrInsecureWithCert)
	})

	t.Run("invalid PEM", func(t *testing.T) {
		_, err := transportCredentials([]byte("not a cert"), false)
		assert.ErrorIs(t, err, ErrBadCACert)
	})
}

func TestDial_CertFile(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := Dial("localhost:1", filepath.Join(t.TempDir(), "nope.crt"), false, nil)
		assert.Error(t, err)
	})

	t.Run("valid file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "ca.crt")
		require.NoError(t, os.WriteFile(path, testCAPEM(t), 0600))
		conn, err := Dial("localhost:1", path, false, nil)
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	})
}
