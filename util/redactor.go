package util

import (
	"bytes"
	"net/url"
	"os"
)

const redactReplacement = "***"

var RedactHook = Hook

type Redactor struct {
	redact []string
}

// Redact adds items for redaction
func (l *Redactor) Redact(redact ...string) {
	for _, s := range redact {
		if RedactHook != nil && len(s) > 0 {
			l.redact = append(l.redact, RedactHook(s)...)
		}
	}
}

func (l *Redactor) Write(p []byte) (n int, err error) {
	for _, s := range l.redact {
		p = bytes.ReplaceAll(p, []byte(s), []byte(redactReplacement))
	}
	return os.Stdout.Write(p)
}

func Hook(s string) []string {
	return []string{s, url.QueryEscape(s)}
}
