package util

import (
	"bytes"
	"net/url"
	"os"
	"sync"
)

var (
	// RedactReplacement is the default replacement string
	RedactReplacement = "***"

	// RedactHook is the hook for expanding different representations of
	// redaction items. Setting to nil will disable redaction.
	RedactHook = RedactDefaultHook
)

// Redactor implements a redacting io.Writer
type Redactor struct {
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

func (l *Redactor) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	for _, s := range l.redact {
		p = bytes.ReplaceAll(p, []byte(s), []byte(RedactReplacement))
	}
	l.mu.Unlock()
	return os.Stdout.Write(p)
}

// RedactDefaultHook expands a redaction item to include URL encoding
func RedactDefaultHook(s string) []string {
	return []string{s, url.QueryEscape(s)}
}
