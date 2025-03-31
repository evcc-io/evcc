package log

import (
	"log/slog"
	"os"
	"time"

	slogformatter "github.com/samber/slog-formatter"
)

func init() {
	logger := slog.New(DefaultHandler())
	slog.SetDefault(logger)

	slog.Info("logging initialized")
}

func DefaultHandler() slog.Handler {
	opts := &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		ReplaceAttr: ReplaceAttr,
	}

	return slogformatter.NewFormatterHandler(
		slogformatter.TimeFormatter(time.DateTime, time.Local),
	)(&ContextHandler{
		Handler: slog.NewTextHandler(os.Stdout, opts),
	})
}
