package log

import (
	"context"
	"log/slog"
)

type contextKey struct{}

var ctxAttrsKey contextKey

// ContextHandler adds contextual attributes to the Record before calling the underlying handler
type ContextHandler struct {
	slog.Handler
}

// Handle adds contextual attributes to the Record before calling the underlying handler
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(ctxAttrsKey).([]slog.Attr); ok {
		r.AddAttrs(attrs...)
	}

	return h.Handler.Handle(ctx, r)
}

func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Handler.Enabled(ctx, level)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithGroup(name)}
}

// AppendCtx adds slog attributes to the provided context
func AppendCtx(parent context.Context, attr ...slog.Attr) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	if v, ok := parent.Value(ctxAttrsKey).([]slog.Attr); ok {
		return context.WithValue(parent, ctxAttrsKey, append(v, attr...))
	}

	return context.WithValue(parent, ctxAttrsKey, attr)
}
