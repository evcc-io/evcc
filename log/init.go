package log

import (
	"log/slog"
	"os"
)

func init() {
	opts := &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		ReplaceAttr: ReplaceAttr,
	}

	logger := slog.New(&ContextHandler{
		Handler: slog.NewTextHandler(os.Stdout, opts),
	})

	slog.SetDefault(logger)

	slog.Info("logging initialized")
}
