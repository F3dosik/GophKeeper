package export

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const password = "export-password-1"

func sampleArchive() *Archive {
	return &Archive{
		ExportedAt: time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC),
		Login:      "alice",
		Secrets: []domain.SecretPayload{
			{Name: "github", Type: domain.SecretTypeCredentials, Data: json.RawMessage(`{"login":"me","password":"p@ss"}`), Metadata: "work"},
			{Name: "key.bin", Type: domain.SecretTypeBinary, Data: json.RawMessage(`{"data":"AAEC/w=="}`)},
		},
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	file, err := Encrypt(sampleArchive(), password)
	require.NoError(t, err)

	// Содержимое секретов в файле не видно.
	assert.NotContains(t, string(file), "github")
	assert.NotContains(t, string(file), "p@ss")
	assert.Contains(t, string(file), `"format": "gophkeeper-export"`)

	got, err := Decrypt(file, password)
	require.NoError(t, err)
	assert.Equal(t, sampleArchive(), got)
}

func TestEncrypt_SaltIsRandom(t *testing.T) {
	a, err := Encrypt(sampleArchive(), password)
	require.NoError(t, err)
	b, err := Encrypt(sampleArchive(), password)
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
}

func TestEncrypt_WeakPassword(t *testing.T) {
	_, err := Encrypt(sampleArchive(), "short")
	assert.ErrorIs(t, err, ErrWeakPassword)
}

func TestDecrypt_Errors(t *testing.T) {
	file, err := Encrypt(sampleArchive(), password)
	require.NoError(t, err)

	t.Run("wrong password", func(t *testing.T) {
		_, err := Decrypt(file, "another-password")
		assert.ErrorIs(t, err, ErrWrongPassword)
	})

	t.Run("tampered ciphertext", func(t *testing.T) {
		var env envelope
		require.NoError(t, json.Unmarshal(file, &env))
		env.Data[len(env.Data)-1] ^= 0xff
		tampered, _ := json.Marshal(env)
		_, err := Decrypt(tampered, password)
		assert.ErrorIs(t, err, ErrWrongPassword)
	})

	t.Run("not an export", func(t *testing.T) {
		for _, f := range []string{"", "not json", `{"format":"something-else"}`} {
			_, err := Decrypt([]byte(f), password)
			assert.ErrorIs(t, err, ErrNotExport, f)
		}
	})

	t.Run("newer version", func(t *testing.T) {
		newer := strings.Replace(string(file), `"version": 1`, `"version": 99`, 1)
		_, err := Decrypt([]byte(newer), password)
		assert.ErrorIs(t, err, ErrUnsupportedVersion)
	})

	t.Run("absurd kdf params are rejected before deriving the key", func(t *testing.T) {
		var env envelope
		require.NoError(t, json.Unmarshal(file, &env))
		env.KDF.MemoryKiB = 64 << 20 // 64 GiB
		bad, _ := json.Marshal(env)
		start := time.Now()
		_, err := Decrypt(bad, password)
		assert.ErrorIs(t, err, ErrNotExport)
		assert.Less(t, time.Since(start), time.Second)
	})
}

// fakeSaver — хранилище в памяти; existing — уже имеющиеся секреты (по имени).
type fakeSaver struct {
	existing map[string]bool
	updated  []string
	failOn   string
	fatalOn  string
}

var errFatal = errors.New("session expired")

func (f *fakeSaver) CreateSecret(_ context.Context, p *domain.SecretPayload) error {
	switch {
	case p.Name == f.fatalOn:
		return errFatal
	case p.Name == f.failOn:
		return domain.ErrInvalidArgument
	case f.existing[p.Name]:
		return domain.ErrAlreadyExists
	}
	f.existing[p.Name] = true
	return nil
}

func (f *fakeSaver) UpdateSecret(_ context.Context, p *domain.SecretPayload) error {
	f.updated = append(f.updated, p.Name)
	return nil
}

func archiveOf(names ...string) *Archive {
	a := &Archive{}
	for _, n := range names {
		a.Secrets = append(a.Secrets, domain.SecretPayload{Name: n, Type: domain.SecretTypeText})
	}
	return a
}

func TestImport(t *testing.T) {
	ctx := context.Background()
	isFatal := func(err error) bool { return errors.Is(err, errFatal) }

	t.Run("skip existing by default", func(t *testing.T) {
		s := &fakeSaver{existing: map[string]bool{"b": true}, failOn: "c"}
		report, err := Import(ctx, s, archiveOf("a", "b", "c"), false, isFatal)
		require.NoError(t, err)
		assert.Equal(t, 1, report.Created)
		assert.Equal(t, 1, report.Skipped)
		assert.Equal(t, 0, report.Updated)
		require.Len(t, report.Failed, 1)
		assert.Equal(t, "c", report.Failed[0].Name)
	})

	t.Run("overwrite existing", func(t *testing.T) {
		s := &fakeSaver{existing: map[string]bool{"b": true}}
		report, err := Import(ctx, s, archiveOf("a", "b"), true, isFatal)
		require.NoError(t, err)
		assert.Equal(t, 1, report.Created)
		assert.Equal(t, 1, report.Updated)
		assert.Equal(t, []string{"b"}, s.updated)
	})

	t.Run("fatal error stops with a partial report", func(t *testing.T) {
		s := &fakeSaver{existing: map[string]bool{}, fatalOn: "b"}
		report, err := Import(ctx, s, archiveOf("a", "b", "c"), false, isFatal)
		assert.ErrorIs(t, err, errFatal)
		assert.Equal(t, 1, report.Created, "secrets before the fatal error are imported")
		assert.False(t, s.existing["c"], "nothing after the fatal error")
	})
}
