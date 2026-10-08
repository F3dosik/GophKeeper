package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/F3dosik/GophKeeper/internal/logger"
	"github.com/caarlos0/env/v11"
)

const (
	defaultServerPort = "50051"
	defaultLogLevel   = string(logger.ModeDevelopment)
	defaultTokenTTL   = 24 * time.Hour
)

// Config содержит конфигурацию сервера.
type Config struct {
	DatabaseURL string        `env:"DATABASE_URL"`
	JWTSecret   string        `env:"JWT_SECRET"`
	ServerPort  string        `env:"SERVER_PORT"`
	LogLevel    string        `env:"LOG_LEVEL"`
	TokenTTL    time.Duration `env:"TOKEN_TTL"`
	// TLSCertFile и TLSKeyFile — пути к сертификату и приватному ключу сервера (PEM).
	// Задаются вместе; если оба пусты, сервер работает без TLS (только для локальной разработки).
	TLSCertFile string `env:"TLS_CERT_FILE"`
	TLSKeyFile  string `env:"TLS_KEY_FILE"`
}

// TLSEnabled сообщает, настроен ли TLS.
func (c *Config) TLSEnabled() bool {
	return c.TLSCertFile != "" && c.TLSKeyFile != ""
}

// Load загружает и валидирует конфигурацию из переменных окружения.
func Load() (*Config, error) {
	config, err := parseConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return config, nil
}

func parseConfig() (*Config, error) {
	var config Config
	err := env.Parse(&config)
	if err != nil {
		return nil, fmt.Errorf("parseconfig: %w", err)
	}

	if config.ServerPort == "" {
		config.ServerPort = defaultServerPort
	}

	if !strings.HasPrefix(config.ServerPort, ":") {
		config.ServerPort = ":" + config.ServerPort
	}

	if config.LogLevel == "" {
		config.LogLevel = defaultLogLevel
	}

	if config.TokenTTL == 0 {
		config.TokenTTL = defaultTokenTTL
	}

	return &config, nil
}

// Validate проверяет корректность конфигурации.
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	if (c.TLSCertFile == "") != (c.TLSKeyFile == "") {
		return fmt.Errorf("TLS_CERT_FILE and TLS_KEY_FILE must be set together")
	}

	if c.TokenTTL <= 0 {
		return fmt.Errorf("TOKEN_TTL must be positive")
	}

	switch c.LogLevel {
	case string(logger.ModeDevelopment), string(logger.ModeProduction):
	default:
		return fmt.Errorf("invalid log mode: %s, allowed: development, production", c.LogLevel)
	}

	return nil
}
