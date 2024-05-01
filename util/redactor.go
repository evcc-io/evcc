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

// Redactor implements a redacting io.Writer
type Redactor struct {
	io.Writer
	mu     sync.Mutex
	redact []string
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

func (l *Redactor) redacted(p []byte) []byte {
	l.mu.Lock()
	for _, s := range l.redact {
		p = bytes.ReplaceAll(p, []byte(s), []byte(RedactReplacement))
	}
	l.mu.Unlock()
	return p
}

type redactWriter struct {
	io.Writer
	redactor *Redactor
}

func (w *redactWriter) Write(p []byte) (n int, err error) {
	return w.Writer.Write(w.redactor.redacted(p))
}
