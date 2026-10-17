// Package config содержит конфигурацию клиента Shron и её загрузку
// из переменных окружения.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/F3dosik/Shron/internal/client/grpcclient"
	"github.com/caarlos0/env/v11"
)

const (
	defaultServerAddress = "localhost:50051"
	defaultSessionPath   = "~/.shron/session"
	// legacyDataDir — каталог данных CLI до переименования проекта; переносится
	// в каталог defaultSessionPath при первом запуске.
	legacyDataDir = "~/.gophkeeper"
)

// Config содержит конфигурацию клиента. Вместо SHRON_* принимаются и прежние
// имена GOPHKEEPER_* (см. Getenv).
type Config struct {
	ServerAddress string `env:"SHRON_SERVER"`
	SessionPath   string `env:"SHRON_SESSION"`
	TLSCertPath   string `env:"SHRON_TLS_CERT"`
	// Insecure разрешает соединение без TLS. Только для локальной разработки:
	// токен и ключ аутентификации передаются открытым текстом.
	Insecure bool `env:"SHRON_INSECURE"`
}

// Load загружает и валидирует конфигурацию из переменных окружения.
// Если путь к сессии не задан, переносит каталог данных прежней версии (~/.gophkeeper).
func Load() (*Config, error) {
	var config Config
	if err := env.ParseWithOptions(&config, env.Options{Environment: environment()}); err != nil {
		return nil, fmt.Errorf("Load: failed to parse env config: %w", err)
	}

	if config.ServerAddress == "" {
		config.ServerAddress = defaultServerAddress
	}
	// Порт необязателен: «192.168.1.5» означает «192.168.1.5:50051».
	address, err := grpcclient.NormalizeAddress(config.ServerAddress)
	if err != nil {
		return nil, fmt.Errorf("Load: SHRON_SERVER %q: %w", config.ServerAddress, err)
	}
	config.ServerAddress = address

	if config.SessionPath == "" {
		config.SessionPath = defaultSessionPath
		newDir := filepath.Dir(expandHome(defaultSessionPath))
		if err := MigrateDir(expandHome(legacyDataDir), newDir); err != nil {
			return nil, fmt.Errorf("Load: %w", err)
		}
	}

	config.SessionPath = expandHome(config.SessionPath)

	return &config, nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
