package ocpp

import (
	"testing"
)

func TestExternalUrl(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"", ""},
		{"http://example.com", "ws://example.com:8887"},
		{"https://example.com:443", "ws://example.com:8887"},
		{"http://10.20.30.40:7070/path", "ws://10.20.30.40:8887/path"},
	}

	for _, tt := range tests {
		externalUrl = tt.input
		if result := ExternalUrl(); result != tt.expected {
			t.Errorf("ExternalUrl(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
