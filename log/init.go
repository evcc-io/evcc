package log

import (
	"log/slog"
)

func init() {
	logger := slog.New(DefaultHandler())
	slog.SetDefault(logger)
}

func DefaultHandler() slog.Handler {
	opts := &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		ReplaceAttr: ReplaceAttr,
	}

	return &ContextHandler{
		Handler: NewTextHandlerWrapper(opts),
	}
}
