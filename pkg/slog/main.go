package slog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.SugaredLogger

// Init initializes the logger
func Init() {
	config := zap.NewProductionConfig()

	// Configure the encoder to use ISO8601 time format
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, _ := config.Build()
	log = logger.Sugar()
}

// Get returns the global logger instance
func Get() *zap.SugaredLogger {
	return log
}

// Sync flushes any buffered log entries
func Sync() error {
	return log.Sync()
}
