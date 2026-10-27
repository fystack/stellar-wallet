// Package logger builds the process-wide zap logger from config.
package logger

import (
	"fmt"

	"stellar-wallet-backend/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New returns a colored console logger for local use or a JSON logger for production.
func New(cfg config.LogConfig) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}
	var zapConfig zap.Config
	if cfg.Format == "json" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")
		zapConfig.DisableStacktrace = true
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)
	return zapConfig.Build()
}
