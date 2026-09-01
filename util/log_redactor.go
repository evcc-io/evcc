package util

import (
	"bytes"
	"io"
	"net/url"
	"sync"
)

var (
	// RedactReplacement is the default replacement string
	RedactReplacement = "***"

	// RedactHook is the hook for expanding different representations of
	// redaction items. Setting to nil will disable redaction.
	RedactHook = RedactDefaultHook
)

// RedactDefaultHook expands a redaction item to include URL encoding
func RedactDefaultHook(s string) []string {
	return []string{s, url.QueryEscape(s)}
}

// maxRotatingSlots limits the number of rotating redaction slots per logger.
// Loggers are shared per log area, so repeatedly re-created token sources would
// otherwise keep reserving slots. Beyond the limit slots are reused in order,
// evicting the oldest.
const maxRotatingSlots = 32

// Redactor implements log redaction
type Redactor struct {
	mu       sync.Mutex
	redact   []string
	rotating [][]string
	nextSlot int
}

// Redact adds items for redaction
func (l *Redactor) Redact(redact ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, s := range redact {
		if RedactHook != nil && len(s) > 0 {
			l.redact = append(l.redact, RedactHook(s)...)
		}
	}
}

// RotatingSlot reserves a redaction slot for periodically refreshed secrets
// like access and refresh tokens. Updating the slot replaces the previous
// values instead of appending, so the redaction list does not grow with every
// refresh.
func (l *Redactor) RotatingSlot() func(...string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	idx := len(l.rotating)
	if idx < maxRotatingSlots {
		l.rotating = append(l.rotating, nil)
	} else {
		idx = l.nextSlot
		l.nextSlot = (l.nextSlot + 1) % maxRotatingSlots
		l.rotating[idx] = nil
	}

	return func(s ...string) {
		l.mu.Lock()
		defer l.mu.Unlock()

		l.rotating[idx] = nil

		if RedactHook == nil {
			return
		}

		for _, v := range s {
			if len(v) > 0 {
				l.rotating[idx] = append(l.rotating[idx], RedactHook(v)...)
			}
		}
	}
}

func (l *Redactor) redacted(p []byte) []byte {
	l.mu.Lock()
	for _, s := range l.redact {
		p = bytes.ReplaceAll(p, []byte(s), []byte(RedactReplacement))
	}
	for _, slot := range l.rotating {
		for _, s := range slot {
			p = bytes.ReplaceAll(p, []byte(s), []byte(RedactReplacement))
		}
	}
	l.mu.Unlock()
	return p
}

// redactWriter implements a redacting io.Writer
type redactWriter struct {
	w io.Writer
	r *Redactor
}

func (w *redactWriter) Write(p []byte) (n int, err error) {
	return w.w.Write(w.r.redacted(p))
}
