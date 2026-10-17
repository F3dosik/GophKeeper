package domain

import (
	"strings"
)

// AndroidAppURIPrefix помечает в CredentialsSecret.URIs приложение Android по имени
// пакета: «androidapp://com.github.android». Без префикса «github.com» и
// «com.github.android» не различить.
const AndroidAppURIPrefix = "androidapp://"

// MaxCredentialURIs — сколько сайтов и приложений можно привязать к одному логину.
const MaxCredentialURIs = 20

// ParseURIs разбирает список сайтов и приложений, введённый одной строкой: элементы
// разделяются переводами строк, запятыми или пробелами. Пустые элементы и повторы
// отбрасываются, порядок сохраняется. Сами значения не проверяются: это адрес, домен
// или приложение, как их ввёл пользователь; сравнивает их клиент, который заполняет формы.
func ParseURIs(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == ' ' || r == '\t'
	})
	var uris []string
	seen := make(map[string]bool, len(fields))
	for _, f := range fields {
		key := strings.ToLower(f)
		if seen[key] {
			continue
		}
		seen[key] = true
		uris = append(uris, f)
		if len(uris) == MaxCredentialURIs {
			break
		}
	}
	return uris
}
