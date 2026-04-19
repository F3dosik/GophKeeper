package app_test

import (
	"testing"

	"github.com/F3dosik/GophKeeper/internal/server/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("LOG_LEVEL", "")
}

func TestLoad_Defaults(t *testing.T) {
	setValidEnv(t)

	cfg, err := app.Load()

	require.NoError(t, err)
	assert.Equal(t, ":50051", cfg.ServerPort)
	assert.Equal(t, "development", cfg.LogLevel)
	assert.Equal(t, "postgres://localhost/db", cfg.DatabaseURL)
	assert.Equal(t, "secret", cfg.JWTSecret)
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("JWT_SECRET", "s")
	t.Setenv("SERVER_PORT", "8080")
	t.Setenv("LOG_LEVEL", "production")

	cfg, err := app.Load()

	require.NoError(t, err)
	assert.Equal(t, ":8080", cfg.ServerPort, "port without colon should be prefixed")
	assert.Equal(t, "production", cfg.LogLevel)
}

func TestLoad_PortWithColonUnchanged(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("JWT_SECRET", "s")
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
		{"missing DATABASE_URL", app.Config{ServerPort: ":50051", JWTSecret: "s", LogLevel: "development"}, "DATABASE_URL"},
		{"missing JWT_SECRET", app.Config{ServerPort: ":50051", DatabaseURL: "postgres://", LogLevel: "development"}, "JWT_SECRET"},
		{"invalid log level", app.Config{ServerPort: ":50051", DatabaseURL: "postgres://", JWTSecret: "s", LogLevel: "debug"}, "invalid log mode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestValidate_Success(t *testing.T) {
	cfg := app.Config{
		ServerPort:  ":50051",
		DatabaseURL: "postgres://",
		JWTSecret:   "s",
		LogLevel:    "production",
	}
	assert.NoError(t, cfg.Validate())
}

func TestLoad_ValidationErrorPropagates(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "s")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("LOG_LEVEL", "")

	_, err := app.Load()
	assert.Error(t, err)
}
