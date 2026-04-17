package simplelog

import (
	"context"

	"go.uber.org/zap"
)

type SimpleLogKey struct{}

var SimpleLogKeyCtx = SimpleLogKey{}

type SimpleZapLogger struct {
	*zap.Logger
}

func NewSimpleZapLogger(logger *zap.Logger) *SimpleZapLogger {
	return &SimpleZapLogger{logger}
}

func (logger *SimpleZapLogger) WithContext(ctx context.Context) *SimpleZapLogger {
	if fields, ok := ctx.Value(SimpleLogKeyCtx).([]zap.Field); ok {
		logger.Logger = logger.With(fields...)
	}
	return logger
}
