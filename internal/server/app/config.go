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
	// Короткий срок жизни ограничивает ущерб от украденного токена; клиент получает
	// новый токен при каждой проверке мастер-пароля, поэтому частый вход не нужен.
	defaultTokenTTL = time.Hour

	defaultAuthRateLimit  = 30      // запросов к Auth в минуту с одного IP
	defaultAuthRateBurst  = 20      // запросов к Auth подряд с одного IP (10 операций с секретами)
	defaultSecretMaxSize  = 1 << 20 // 1 MiB зашифрованных данных на секрет
	defaultSecretMaxCount = 1000    // секретов на пользователя
	maxSecretMaxSize      = 32 << 20

	defaultTempPasswordTTL = 24 * time.Hour
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

	// AuthRateLimit и AuthRateBurst ограничивают частоту вызовов GetSalt/CreateUser/Login
	// с одного IP: среднее число запросов в минуту и максимум запросов подряд.
	AuthRateLimit int `env:"AUTH_RATE_LIMIT"`
	AuthRateBurst int `env:"AUTH_RATE_BURST"`

	// SecretMaxSize — максимальный размер зашифрованных данных одного секрета в байтах,
	// SecretMaxCount — максимальное количество секретов у пользователя.
	SecretMaxSize  int `env:"SECRET_MAX_SIZE"`
	SecretMaxCount int `env:"SECRET_MAX_COUNT"`

	// RegistrationEnabled разрешает регистрацию на публичном порту (по умолчанию true).
	RegistrationEnabled bool `env:"REGISTRATION_ENABLED" envDefault:"true"`
	// AdminPort — порт административного сервера, на котором регистрация разрешена
	// всегда, включая временные пароли. Пустой — административный сервер не запускается.
	// Порт должен быть доступен только с машины сервера.
	AdminPort string `env:"ADMIN_PORT"`
	// TempPasswordTTL — срок действия временного пароля.
	TempPasswordTTL time.Duration `env:"TEMP_PASSWORD_TTL"`
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

	if config.AuthRateLimit == 0 {
		config.AuthRateLimit = defaultAuthRateLimit
	}

	if config.AuthRateBurst == 0 {
		config.AuthRateBurst = defaultAuthRateBurst
	}

	if config.SecretMaxSize == 0 {
		config.SecretMaxSize = defaultSecretMaxSize
	}

	if config.SecretMaxCount == 0 {
		config.SecretMaxCount = defaultSecretMaxCount
	}

	if config.AdminPort != "" && !strings.HasPrefix(config.AdminPort, ":") {
		config.AdminPort = ":" + config.AdminPort
	}

	if config.TempPasswordTTL == 0 {
		config.TempPasswordTTL = defaultTempPasswordTTL
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

	if c.AuthRateLimit <= 0 || c.AuthRateBurst <= 0 {
		return fmt.Errorf("AUTH_RATE_LIMIT and AUTH_RATE_BURST must be positive")
	}

	if c.SecretMaxSize <= 0 || c.SecretMaxCount <= 0 {
		return fmt.Errorf("SECRET_MAX_SIZE and SECRET_MAX_COUNT must be positive")
	}

	if c.TempPasswordTTL < 0 {
		return fmt.Errorf("TEMP_PASSWORD_TTL must be positive")
	}

	if c.AdminPort != "" && c.AdminPort == c.ServerPort {
		return fmt.Errorf("ADMIN_PORT must differ from SERVER_PORT")
	}

	// Клиент принимает ответы до 64 MiB; страница списка содержит хотя бы один секрет.
	if c.SecretMaxSize > maxSecretMaxSize {
		return fmt.Errorf("SECRET_MAX_SIZE must not exceed %d bytes", maxSecretMaxSize)
	}

	return nil
}
