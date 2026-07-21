package util

import (
	"context"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/util/logstash"
)

// handler implements slog.Handler. It renders the evcc log line format, applies redaction
// and fans out to stdout (gated by area level), the logstash ring and the ui (warn+).
type handler struct {
	padded   string
	level    *slog.LevelVar // stdout threshold
	redactor *Redactor
	lp       int
	attrs    []slog.Attr
	groups   []string
}

func (h *handler) Enabled(_ context.Context, _ slog.Level) bool {
	// all records reach the logstash ring buffer; stdout is gated in Handle
	return true
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.withAttrs(attrs)
}

func (h *handler) WithGroup(name string) slog.Handler {
	c := *h
	c.groups = append(slices.Clip(h.groups), name)
	return &c
}

func (h *handler) withAttrs(attrs []slog.Attr) *handler {
	c := *h
	c.attrs = append(slices.Clip(h.attrs), attrs...)
	return &c
}

// with converts loosely-typed slog arguments into attributes
func (h *handler) with(args []any) *handler {
	var attrs []slog.Attr

	for len(args) > 0 {
		switch a := args[0].(type) {
		case slog.Attr:
			attrs = append(attrs, a)
			args = args[1:]
		case string:
			if len(args) == 1 {
				attrs = append(attrs, slog.String("!BADKEY", a))
				args = nil
				break
			}
			attrs = append(attrs, slog.Any(a, args[1]))
			args = args[2:]
		default:
			attrs = append(attrs, slog.Any("!BADKEY", a))
			args = args[1:]
		}
	}

	return h.withAttrs(attrs)
}

func (h *handler) Handle(_ context.Context, r slog.Record) error {
	text := new(strings.Builder)
	text.WriteString(r.Message)

	for _, a := range h.attrs {
		h.appendAttr(text, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		h.appendAttr(text, a)
		return true
	})

	line := new(strings.Builder)
	line.WriteByte('[')
	line.WriteString(h.padded)
	line.WriteString("] ")
	line.WriteString(levelString(r.Level))
	line.WriteByte(' ')
	line.WriteString(r.Time.Format("2006/01/02 15:04:05"))
	line.WriteByte(' ')
	line.WriteString(text.String())
	line.WriteByte('\n')

	b := h.redactor.redacted([]byte(line.String()))

	if _, err := logstash.DefaultHandler.Write(b); err != nil {
		return err
	}

	if r.Level >= h.level.Level() {
		if _, err := os.Stdout.Write(b); err != nil {
			return err
		}
	}

	if r.Level >= slog.LevelWarn {
		level := "warn"
		if r.Level >= slog.LevelError {
			level = "error"
		}
		uiCapture(level, h.lp, string(h.redactor.redacted([]byte(text.String()))))
	}

	return nil
}

func (h *handler) appendAttr(sb *strings.Builder, a slog.Attr) {
	if a.Equal(slog.Attr{}) {
		return
	}

	key := a.Key
	if len(h.groups) > 0 {
		key = strings.Join(h.groups, ".") + "." + key
	}

	val := a.Value.Resolve().String()
	if strings.ContainsAny(val, " \t\n\"=") {
		val = strconv.Quote(val)
	}

	sb.WriteByte(' ')
	sb.WriteString(key)
	sb.WriteByte('=')
	sb.WriteString(val)
}

func levelString(l slog.Level) string {
	switch {
	case l < LevelDebug:
		return "TRACE"
	case l < LevelInfo:
		return "DEBUG"
	case l < LevelWarn:
		return "INFO"
	case l < LevelError:
		return "WARN"
	case l < LevelFatal:
		return "ERROR"
	default:
		return "FATAL"
	}
}
