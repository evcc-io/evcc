package util

import (
	"context"
)

var CtxLogger = struct{}{}

func WithLogger(ctx context.Context, log *Logger) context.Context {
	return context.WithValue(ctx, CtxLogger, log)
}
