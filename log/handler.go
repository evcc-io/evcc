package log

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"sync"
	"time"
)

const (
	timeFormat = time.DateTime

	reset = "\033[0m"

	black        = 30
	red          = 31
	green        = 32
	yellow       = 33
	blue         = 34
	magenta      = 35
	cyan         = 36
	lightGray    = 37
	darkGray     = 90
	lightRed     = 91
	lightGreen   = 92
	lightYellow  = 93
	lightBlue    = 94
	lightMagenta = 95
	lightCyan    = 96
	white        = 97
)

type thw struct {
	h slog.Handler
	b *bytes.Buffer
	m *sync.Mutex
}

func NewTextHandlerWrapper(opts *slog.HandlerOptions) slog.Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	b := new(bytes.Buffer)

	return &thw{
		h: slog.NewTextHandler(b, &slog.HandlerOptions{
			Level:       opts.Level,
			AddSource:   opts.AddSource,
			ReplaceAttr: suppressDefaults(opts.ReplaceAttr),
		}),
		b: b,
		m: new(sync.Mutex),
	}
}

func (h *thw) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

func (h *thw) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &thw{h: h.h.WithAttrs(attrs), b: h.b, m: h.m}
}

func (h *thw) WithGroup(name string) slog.Handler {
	return &thw{h: h.h.WithGroup(name), b: h.b, m: h.m}
}

func (h *thw) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String()

	switch r.Level {
	case LevelTrace:
		level = colorize(lightGray, level)
	case slog.LevelDebug:
		level = colorize(darkGray, level)
	case slog.LevelInfo:
		level = colorize(cyan, level)
	case slog.LevelWarn:
		level = colorize(lightYellow, level)
	case slog.LevelError:
		level = colorize(lightRed, level)
	}

	attrs, err := h.computeAttrs(ctx, r)
	if err != nil {
		return err
	}

	fmt.Print(
		colorize(lightGray, r.Time.Format(timeFormat)),
		" ",
		level,
		" ",
		colorize(white, r.Message),
		colorize(darkGray, attrs),
	)

	return nil
}

func (h *thw) computeAttrs(ctx context.Context, r slog.Record) (string, error) {
	h.m.Lock()
	defer func() {
		h.b.Reset()
		h.m.Unlock()
	}()

	if err := h.h.Handle(ctx, r); err != nil {
		return "", err
	}

	return h.b.String(), nil
}

func colorize(colorCode int, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorCode), v, reset)
}

func suppressDefaults(next func([]string, slog.Attr) slog.Attr) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if slices.Contains([]string{slog.TimeKey, slog.LevelKey, slog.MessageKey}, a.Key) {
			return slog.Attr{}
		}
		if next == nil {
			return a
		}
		return next(groups, a)
	}
}
