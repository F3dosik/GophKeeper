package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Префиксы переменных окружения: текущий и до переименования проекта GophKeeper → Shron.
const (
	envPrefix       = "SHRON_"
	legacyEnvPrefix = "GOPHKEEPER_"
)

// Getenv возвращает переменную окружения SHRON_<name>, а если она не задана —
// GOPHKEEPER_<name>, чтобы настройки, сделанные до переименования, продолжали работать.
func Getenv(name string) string {
	if v, ok := os.LookupEnv(envPrefix + name); ok {
		return v
	}
	return os.Getenv(legacyEnvPrefix + name)
}

// environment возвращает окружение процесса, дополненное значениями GOPHKEEPER_*
// для незаданных SHRON_* — для разбора Config без изменения окружения процесса.
func environment() map[string]string {
	result := make(map[string]string)
	for _, kv := range os.Environ() {
		if k, v, ok := strings.Cut(kv, "="); ok {
			result[k] = v
		}
	}
	for k, v := range result {
		if name, ok := strings.CutPrefix(k, legacyEnvPrefix); ok {
			if _, set := result[envPrefix+name]; !set {
				result[envPrefix+name] = v
			}
		}
	}
	return result
}

// MigrateDir переносит каталог данных oldDir, созданный до переименования проекта,
// в newDir. Ничего не делает, если newDir уже существует или oldDir нет. В каталоге
// лежат сессия и запомненные параметры Argon2id (kdfpin): без переноса пришлось бы
// снова входить, а защита от ослабления параметров начиналась бы заново.
func MigrateDir(oldDir, newDir string) error {
	if _, err := os.Lstat(newDir); err == nil || !errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if _, err := os.Lstat(oldDir); err != nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(newDir), 0o700); err != nil {
		return fmt.Errorf("migrate %s: %w", oldDir, err)
	}
	if err := os.Rename(oldDir, newDir); err != nil {
		return fmt.Errorf("migrate %s to %s: %w", oldDir, newDir, err)
	}
	return nil
}
