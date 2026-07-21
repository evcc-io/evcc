package util

import (
	"context"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/util/logstash"
)

// handler implements slog.Handler. It applies redaction and fans out structured
// records to stdout (gated by area level), the logstash ring and the ui (warn+).
type handler struct {
	area     string
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

func (h *handler) redact(s string) string {
	return string(h.redactor.redacted([]byte(s)))
}

func (h *handler) Handle(_ context.Context, r slog.Record) error {
	msg := h.redact(r.Message)

	text := new(strings.Builder)
	text.WriteString(msg)

	var attrs map[string]string
	appendAttr := func(a slog.Attr) {
		if a.Equal(slog.Attr{}) {
			return
		}

		key := a.Key
		if len(h.groups) > 0 {
			key = strings.Join(h.groups, ".") + "." + key
		}
		val := h.redact(a.Value.Resolve().String())

		if attrs == nil {
			attrs = make(map[string]string)
		}
		attrs[key] = val

		text.WriteByte(' ')
		text.WriteString(key)
		text.WriteByte('=')
		text.WriteString(logstash.QuoteAttr(val))
	}

	for _, a := range h.attrs {
		appendAttr(a)
	}
	r.Attrs(func(a slog.Attr) bool {
		appendAttr(a)
		return true
	})

	if h.area != "cache" {
		logstash.DefaultHandler.Add(logstash.Entry{
			Time:    r.Time,
			Area:    h.area,
			Level:   r.Level,
			Message: msg,
			Attrs:   attrs,
		})
	}

	if r.Level >= h.level.Level() {
		line := new(strings.Builder)
		line.WriteByte('[')
		line.WriteString(h.padded)
		line.WriteString("] ")
		line.WriteString(logstash.LevelString(r.Level))
		line.WriteByte(' ')
		line.WriteString(r.Time.Format("2006/01/02 15:04:05"))
		line.WriteByte(' ')
		line.WriteString(text.String())
		line.WriteByte('\n')

		if _, err := os.Stdout.Write([]byte(line.String())); err != nil {
			return err
		}
	}

	if r.Level >= slog.LevelWarn {
		level := "warn"
		if r.Level >= slog.LevelError {
			level = "error"
		}
		uiCapture(level, h.lp, text.String())
	}

	return nil
}
