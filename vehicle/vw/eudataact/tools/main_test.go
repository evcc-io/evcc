package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedact(t *testing.T) {
	assert.Equal(t, "login failed: [REDACTED]", redact("login failed: user@example.com", "user@example.com"))
	assert.Equal(t, "[REDACTED] [REDACTED]", redact("secret secret", "secret", ""))
}
