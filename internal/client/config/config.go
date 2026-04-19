// Package config содержит конфигурацию клиента GophKeeper и её загрузку
// из переменных окружения.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v11"
)

const (
	defaultServerAddress = "localhost:50051"
	defaultSessionPath   = "~/.gophkeeper/session"
)

// Config содержит конфигурацию клиента.
type Config struct {
	ServerAddress string `env:"GOPHKEEPER_SERVER"`
	SessionPath   string `env:"GOPHKEEPER_SESSION"`
	TLSCertPath   string `env:"GOPHKEEPER_TLS_CERT"`
}

// Load загружает и валидирует конфигурацию из переменных окружения.
func Load() (*Config, error) {
	var config Config
	if err := env.Parse(&config); err != nil {
		return nil, fmt.Errorf("Load: failed to parse env config: %w", err)
	}

	if config.ServerAddress == "" {
		config.ServerAddress = defaultServerAddress
	}

	if config.SessionPath == "" {
		config.SessionPath = defaultSessionPath
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
