package logger_test

import (
	"testing"

	"github.com/F3dosik/GophKeeper/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestNew_Development(t *testing.T) {
	log := logger.New(logger.ModeDevelopment)
	assert.NotNil(t, log)
	// Не должно паниковать.
	log.Info("dev log")
}

func TestNew_Production(t *testing.T) {
	log := logger.New(logger.ModeProduction)
	assert.NotNil(t, log)
	log.Info("prod log")
}

func TestNew_DefaultFallsBackToDevelopment(t *testing.T) {
	log := logger.New(logger.Mode("unknown"))
	assert.NotNil(t, log)
	log.Info("fallback log")
}
