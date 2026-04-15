package app

import (
	"fmt"
	"log"
	"strings"

	"github.com/F3dosik/GophKeeper/internal/logger"
	"github.com/caarlos0/env/v11"
)

const (
	defaultServerPort = "50051"
	defaultLogLevel   = string(logger.ModeDevelopment)
)

// Config содержит конфигурацию сервера.
type Config struct {
	DatabaseURL string `env:"DATABASE_URL"`
	JWTSecret   string `env:"JWT_SECRET"`
	ServerPort  string `env:"SERVER_PORT"`
	LogLevel    string `env:"LOG_LEVEL"`
}

// Load загружает и валидирует конфигурацию из переменных окружения.
func Load() (*Config, error) {
	config := parseConfig()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return config, nil
}

func parseConfig() *Config {
	var config Config
	err := env.Parse(&config)
	if err != nil {
		log.Printf("Warning: failed to parse env config: %v\n", err)
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

	return &config
}

// Validate проверяет корректность конфигурации.
func (c *Config) Validate() error {
	if c.ServerPort == "" {
		return fmt.Errorf("SERVER_PORT is required")
	}

	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	switch c.LogLevel {
	case string(logger.ModeDevelopment), string(logger.ModeProduction):
	default:
		return fmt.Errorf("invalid log mode: %s, allowed: development, production", c.LogLevel)
	}

	return nil
}
