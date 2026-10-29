package util

import (
	"context"
)

var CtxLogger = struct{}{}

func WithLogger(ctx context.Context, log *Logger) context.Context {
	return context.WithValue(ctx, CtxLogger, log)
}

func contextLogger(ctx context.Context) *Logger {
	if ctx != nil {
		if l, ok := ctx.Value(CtxLogger).(*Logger); ok {
			return l
		}
	}

	return nil
}

// LoggerFromContext returns the context logger with its component attribute extended
// by the given subtype (e.g. charger -> charger/abb), or a plain logger with the
// subtype as area if the context carries no logger.
func LoggerFromContext(ctx context.Context, sub string) *Logger {
	if l := contextLogger(ctx); l != nil {
		return newHandlerLogger(l.handler.withComponentSubtype(sub))
	}

	return NewLogger(sub)
}

// PluginLoggerFromContext returns the context logger with the plugin attribute added,
// or a plain logger with the given area if the context carries no logger.
func PluginLoggerFromContext(ctx context.Context, area string) *Logger {
	if l := contextLogger(ctx); l != nil {
		return l.With(PluginKey, area)
	}

	return NewLogger(area)
}
