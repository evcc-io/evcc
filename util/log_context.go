package util

import (
	"context"
)

var CtxLogger = struct{}{}

func WithLogger(ctx context.Context, log *Logger) context.Context {
	return context.WithValue(ctx, CtxLogger, log)
}

func ContextLogger(ctx context.Context) *Logger {
	if ctx != nil {
		if l, ok := ctx.Value(CtxLogger).(*Logger); ok {
			return l
		}
	}

	return nil
}

func ContextLoggerWithDefault(ctx context.Context, log *Logger) *Logger {
	if log := ContextLogger(ctx); log != nil {
		return log
	}

	return log
}
