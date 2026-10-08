package app_test

import (
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/internal/server/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testJWTSecret = "01234567890123456789012345678901"

func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("SERVER_PORT", "")
	t.Setenv("LOG_LEVEL", "")
	for _, key := range []string{"AUTH_RATE_LIMIT", "AUTH_RATE_BURST", "SECRET_MAX_SIZE", "SECRET_MAX_COUNT",
		"REGISTRATION_ENABLED", "ADMIN_PORT", "TEMP_PASSWORD_TTL"} {
		t.Setenv(key, "")
	}
}

func TestLoad_Defaults(t *testing.T) {
	setValidEnv(t)

	cfg, err := app.Load()

	require.NoError(t, err)
	assert.Equal(t, ":50051", cfg.ServerPort)
	assert.Equal(t, "development", cfg.LogLevel)
	assert.Equal(t, "postgres://localhost/db", cfg.DatabaseURL)
	assert.Equal(t, testJWTSecret, cfg.JWTSecret)
	assert.Equal(t, time.Hour, cfg.TokenTTL)
	assert.Equal(t, 30, cfg.AuthRateLimit)
	assert.Equal(t, 20, cfg.AuthRateBurst)
	assert.Equal(t, 1<<20, cfg.SecretMaxSize)
	assert.Equal(t, 1000, cfg.SecretMaxCount)
	assert.True(t, cfg.RegistrationEnabled, "registration is open by default")
	assert.Empty(t, cfg.AdminPort, "admin server is off by default")
	assert.Equal(t, 24*time.Hour, cfg.TempPasswordTTL)
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("SERVER_PORT", "8080")
	t.Setenv("LOG_LEVEL", "production")

	cfg, err := app.Load()

	require.NoError(t, err)
	assert.Equal(t, ":8080", cfg.ServerPort, "port without colon should be prefixed")
	assert.Equal(t, "production", cfg.LogLevel)
}

func TestLoad_PortWithColonUnchanged(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("SERVER_PORT", ":9090")
	t.Setenv("LOG_LEVEL", "")

	cfg, err := app.Load()

	require.NoError(t, err)
	assert.Equal(t, ":9090", cfg.ServerPort)
}

func TestValidate_Errors(t *testing.T) {
	tests := []struct {
		name string
		cfg  app.Config
		want string
	}{
		{"missing DATABASE_URL", app.Config{ServerPort: ":50051", JWTSecret: testJWTSecret, LogLevel: "development", TokenTTL: time.Hour}, "DATABASE_URL"},
		{"missing JWT_SECRET", app.Config{ServerPort: ":50051", DatabaseURL: "postgres://", LogLevel: "development", TokenTTL: time.Hour}, "JWT_SECRET"},
		{"invalid log level", app.Config{ServerPort: ":50051", DatabaseURL: "postgres://", JWTSecret: testJWTSecret, LogLevel: "debug", TokenTTL: time.Hour}, "invalid log mode"},
		{"non-positive TOKEN_TTL", app.Config{ServerPort: ":50051", DatabaseURL: "postgres://", JWTSecret: "01234567890123456789012345678901", LogLevel: "development", TokenTTL: 0}, "TOKEN_TTL"},
		{"TLS cert without key", app.Config{ServerPort: ":50051", DatabaseURL: "postgres://", JWTSecret: "01234567890123456789012345678901", LogLevel: "development", TokenTTL: time.Hour, TLSCertFile: "cert.pem"}, "TLS_CERT_FILE and TLS_KEY_FILE"},
		{"TLS key without cert", app.Config{ServerPort: ":50051", DatabaseURL: "postgres://", JWTSecret: "01234567890123456789012345678901", LogLevel: "development", TokenTTL: time.Hour, TLSKeyFile: "key.pem"}, "TLS_CERT_FILE and TLS_KEY_FILE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

// validConfig возвращает конфигурацию, проходящую Validate.
func validConfig() app.Config {
	return app.Config{
		ServerPort:     ":50051",
		DatabaseURL:    "postgres://",
		JWTSecret:      testJWTSecret,
		LogLevel:       "production",
		TokenTTL:       time.Hour,
		AuthRateLimit:  30,
		AuthRateBurst:  10,
		SecretMaxSize:  1 << 20,
		SecretMaxCount: 1000,
	}
}

func TestValidate_Success(t *testing.T) {
	cfg := validConfig()
	assert.NoError(t, cfg.Validate())
}

func TestValidate_Limits(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*app.Config)
		want   string
	}{
		{"zero rate limit", func(c *app.Config) { c.AuthRateLimit = 0 }, "AUTH_RATE_LIMIT"},
		{"negative burst", func(c *app.Config) { c.AuthRateBurst = -1 }, "AUTH_RATE_BURST"},
		{"zero secret size", func(c *app.Config) { c.SecretMaxSize = 0 }, "SECRET_MAX_SIZE"},
		{"zero secret count", func(c *app.Config) { c.SecretMaxCount = 0 }, "SECRET_MAX_COUNT"},
		{"secret size above client limit", func(c *app.Config) { c.SecretMaxSize = 33 << 20 }, "must not exceed"},
		{"admin port equals server port", func(c *app.Config) { c.AdminPort = c.ServerPort }, "ADMIN_PORT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.modify(&cfg)
			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestLoad_ValidationErrorPropagates(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "s")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("LOG_LEVEL", "")

	_, err := app.Load()
	assert.Error(t, err)
}

func TestConfig_TLSEnabled(t *testing.T) {
	assert.False(t, (&app.Config{}).TLSEnabled())
	assert.True(t, (&app.Config{TLSCertFile: "cert.pem", TLSKeyFile: "key.pem"}).TLSEnabled())
}

func TestLoad_RegistrationSettings(t *testing.T) {
	setValidEnv(t)
	t.Setenv("REGISTRATION_ENABLED", "false")
	t.Setenv("ADMIN_PORT", "50052")
	t.Setenv("TEMP_PASSWORD_TTL", "2h")

	cfg, err := app.Load()
	require.NoError(t, err)
	assert.False(t, cfg.RegistrationEnabled)
	assert.Equal(t, ":50052", cfg.AdminPort)
	assert.Equal(t, 2*time.Hour, cfg.TempPasswordTTL)
}
