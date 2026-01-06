package redact

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // Strings that should appear
		redacted []string // Strings that should NOT appear
	}{
		{
			name: "redact password",
			input: `site:
  title: My Home
  meters:
    grid: my-grid
user: testuser
password: secretpass123`,
			expected: []string{"password: *****", "title: My Home", "grid: my-grid"},
			redacted: []string{"secretpass123"},
		},
		{
			name: "redact params marked as private",
			input: `vehicle:
  vin: W1234567890123456
  user: john@example.com
  capacity: 50`,
			expected: []string{"vin: *****", "user: *****", "capacity: 50"},
			redacted: []string{"W1234567890123456", "john@example.com"},
		},
		{
			name: "redact multiple sensitive fields",
			input: `config:
  lat: 52.520008
  lon: 13.404954
  zip: 10115
  sponsortoken: abc123
  apikey: xyz789`,
			expected: []string{"lat: *****", "lon: *****", "zip: *****", "sponsortoken: *****", "apikey: *****"},
			redacted: []string{"52.520008", "13.404954", "10115", "abc123", "xyz789"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := String(tt.input)

			// Check expected strings are present
			for _, exp := range tt.expected {
				assert.Contains(t, result, exp, "Expected to find %q in result", exp)
			}

			// Check redacted strings are NOT present
			for _, red := range tt.redacted {
				assert.NotContains(t, result, red, "Expected %q to be redacted", red)
			}
		})
	}
}

func TestMap(t *testing.T) {
	input := map[string]any{
		"title":    "My Home",
		"password": "secret123",
		"user":     "john@example.com",
		"vin":      "W1234567890",
		"capacity": 50,
		"apikey":   "abc-def-123",
	}

	result := Map(input)

	// Check non-sensitive fields are unchanged
	assert.Equal(t, "My Home", result["title"], "title should not be redacted")
	assert.Equal(t, 50, result["capacity"], "capacity should not be redacted")

	// Check sensitive fields are redacted
	sensitiveFields := []string{"password", "user", "vin", "apikey"}
	for _, field := range sensitiveFields {
		assert.Equal(t, "*****", result[field], "%s should be redacted", field)
	}
}
