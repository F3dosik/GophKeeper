// Package logger предоставляет утилиты для инициализации логгера.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Mode определяет режим работы логгера.
type Mode string

const (
	// ModeDevelopment режим разработки с цветным выводом.
	ModeDevelopment Mode = "development"

	// ModeProduction production режим с JSON форматом.
	ModeProduction Mode = "production"
)

// New создаёт новый экземпляр SugaredLogger.
// В development использует цветной вывод.
// В production использует JSON формат.
func New(mode Mode) *zap.SugaredLogger {
	var cfg zap.Config

	switch mode {
	case ModeProduction:
		cfg = zap.NewProductionConfig()
	case ModeDevelopment:
		fallthrough
	default:
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := cfg.Build()
	if err != nil {
		fallback, _ := zap.NewDevelopment()
		fallback.Fatal("failed to initialize zap logger", zap.Error(err))
	}

	return logger.Sugar()
}
