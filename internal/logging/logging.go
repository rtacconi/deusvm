package logging

import (
	"go.uber.org/zap"
)

// Field is a convenience alias so callers do not import zap directly from most packages.
func Field(key string, value any) zap.Field { return zap.Any(key, value) }

// FieldError standardizes error fields.
func FieldError(err error) zap.Field { return zap.Error(err) }

// New creates a production-ready logger for the daemon.
func New() *zap.Logger {
	logger, _ := zap.NewProduction()
	return logger
}
