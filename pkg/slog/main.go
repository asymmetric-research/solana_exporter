package slog

import (
	"go.uber.org/zap"
)

var log *zap.SugaredLogger

// Init initializes the logger
func Init() {
	logger, _ := zap.NewProduction()
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
