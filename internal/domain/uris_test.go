package domain_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/F3dosik/Shron/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseURIs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"only separators", " ,\n; \t", nil},
		{"one per line", "github.com\nandroidapp://com.github.android\n", []string{"github.com", "androidapp://com.github.android"}},
		{"commas and spaces", "github.com, gitlab.com  bitbucket.org", []string{"github.com", "gitlab.com", "bitbucket.org"}},
		{"windows line breaks", "a.ru\r\nb.ru", []string{"a.ru", "b.ru"}},
		{"duplicates ignoring case", "GitHub.com github.com https://github.com/login", []string{"GitHub.com", "https://github.com/login"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, domain.ParseURIs(tt.in))
		})
	}
}

func TestParseURIs_Limit(t *testing.T) {
	var b strings.Builder
	for i := 0; i < domain.MaxCredentialURIs+5; i++ {
		fmt.Fprintf(&b, "site%d.com\n", i)
	}
	assert.Len(t, domain.ParseURIs(b.String()), domain.MaxCredentialURIs)
}

func TestCredentialsSecret_URIsAreOptional(t *testing.T) {
	// Секреты, созданные до появления поля, читаются без него.
	var old domain.CredentialsSecret
	require.NoError(t, json.Unmarshal([]byte(`{"login":"me","password":"pw"}`), &old))
	assert.Empty(t, old.URIs)

	// Без сайтов поле не пишется: данные таких секретов не меняются.
	data, err := json.Marshal(domain.CredentialsSecret{Login: "me", Password: "pw"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"login":"me","password":"pw"}`, string(data))

	data, err = json.Marshal(domain.CredentialsSecret{Login: "me", Password: "pw", URIs: []string{"github.com"}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"login":"me","password":"pw","uris":["github.com"]}`, string(data))
}
