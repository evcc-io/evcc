package redact

import (
	"net/url"
	"os"
	"strings"
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

func (l *Redactor) Safe(p string) string {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, s := range l.redact {
		p = strings.ReplaceAll(p, s, RedactReplacement)
	}

	return p
}

func (l *Redactor) Write(p []byte) (n int, err error) {
	return os.Stdout.Write([]byte(l.Safe(string(p))))
}

// RedactDefaultHook expands a redaction item to include URL encoding
func RedactDefaultHook(s string) []string {
	return []string{s, url.QueryEscape(s)}
}
