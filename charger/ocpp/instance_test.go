package ocpp

import (
	"testing"
)

func TestExternalUrl(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"", ""},
		{"http://example.com:7070", "ws://example.com:8887"},
		{"https://example.com:443", "wss://example.com:8887"},
		{"http://example.com", "ws://example.com:8887"},
		{"https://example.com", "wss://example.com:8887"},
		{"http://10.20.30.40:7070/path", "ws://10.20.30.40:8887/path"},
		{"https://example.com/path", "wss://example.com:8887/path"},
		{"ws://example.com", "ws://example.com:8887"},
		{"wss://example.com:9000", "wss://example.com:8887"},
		{"strange://example.com", "strange://example.com:8887"},
	}

	for _, tt := range tests {
		externalUrl = tt.input
		if result := ExternalUrl(); result != tt.expected {
			t.Errorf("ExternalUrl(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
