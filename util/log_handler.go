package util

import (
	"context"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/util/logstash"
)

// slogOutput switches stdout to the slog default text format (opt-in via SLOG=1)
var slogOutput = os.Getenv("SLOG") != ""

var textHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	Level: logstash.LevelTrace,
	ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.LevelKey {
			if l, ok := a.Value.Any().(slog.Level); ok {
				a.Value = slog.StringValue(logstash.LevelString(l))
			}
		}
		return a
	},
})

// handler implements slog.Handler. It applies redaction and fans out structured
// records to stdout (gated by area level), the logstash ring and the ui (warn+).
type handler struct {
	area     string
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

// withComponentSubtype extends the component attribute by a subtype, e.g. charger -> charger/abb
func (h *handler) withComponentSubtype(sub string) *handler {
	c := *h
	c.attrs = slices.Clone(h.attrs)
	for i, a := range c.attrs {
		if a.Key == ComponentKey {
			c.attrs[i] = slog.String(ComponentKey, a.Value.Resolve().String()+"/"+sub)
			return &c
		}
	}
	c.attrs = append(c.attrs, slog.String(ComponentKey, sub))
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

func (h *handler) Handle(ctx context.Context, r slog.Record) error {
	var attrs map[string]string
	addAttr := func(a slog.Attr) {
		if a.Equal(slog.Attr{}) {
			return
		}

		key := a.Key
		if len(h.groups) > 0 {
			key = strings.Join(h.groups, ".") + "." + key
		}

		if attrs == nil {
			attrs = make(map[string]string)
		}
		attrs[key] = h.redact(a.Value.Resolve().String())
	}

	for _, a := range h.attrs {
		addAttr(a)
	}
	r.Attrs(func(a slog.Attr) bool {
		addAttr(a)
		return true
	})

	e := logstash.Entry{
		Time:    r.Time,
		Area:    h.area,
		Level:   r.Level,
		Message: h.redact(r.Message),
		Attrs:   attrs,
	}

	if h.area != "cache" {
		logstash.DefaultHandler.Add(e)
	}

	if r.Level >= h.level.Level() {
		if slogOutput {
			rr := slog.NewRecord(r.Time, r.Level, e.Message, 0)
			rr.AddAttrs(slog.String("area", h.area))
			for _, k := range slices.Sorted(maps.Keys(e.Attrs)) {
				rr.AddAttrs(slog.String(k, e.Attrs[k]))
			}

			if err := textHandler.Handle(ctx, rr); err != nil {
				return err
			}
		} else if _, err := os.Stdout.Write([]byte(e.String())); err != nil {
			return err
		}
	}

	if r.Level >= slog.LevelWarn {
		level := "warn"
		if r.Level >= slog.LevelError {
			level = "error"
		}
		uiCapture(level, h.lp, e.Text())
	}

	return nil
}
