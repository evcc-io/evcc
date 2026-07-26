package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogArea(t *testing.T) {
	for _, tc := range []struct {
		title    string
		expected string
	}{
		{"", "db:1"},
		{"Garage", "db:1-Garage"},
		{"db:1", "db:1"},
	} {
		c := Config{ID: 1, Properties: Properties{Title: tc.title}}
		assert.Equal(t, tc.expected, c.LogArea())
	}
}
