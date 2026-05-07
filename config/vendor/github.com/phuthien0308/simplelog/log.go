package simplelog

import (
	"context"

	"go.uber.org/zap"
)

type SimpleLogKey struct{}

var SimpleLogKeyCtx = SimpleLogKey{}

type SimpleLogger struct {
	*zap.Logger
}

func NewSimpleLogger(logger *zap.Logger) *SimpleLogger {
	return &SimpleLogger{logger}
}

func (logger *SimpleLogger) WithContext(ctx context.Context) *SimpleLogger {
	if fields, ok := ctx.Value(SimpleLogKeyCtx).([]zap.Field); ok {
		logger.Logger = logger.With(fields...)
	}
	return logger
}
