package cmd

import (
	"testing"

	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
)

func TestNameForConfig(t *testing.T) {
	for _, tc := range []struct {
		title    string
		expected string
	}{
		{"", "db:8"},
		{"Solar Forecast 1. Dachseite", "db:8-Solar Forecast 1. Dachseite"},
		{"db:8", "db:8"}, // title equal to the generated name is not repeated
	} {
		conf := config.Config{ID: 8, Properties: config.Properties{Title: tc.title}}
		assert.Equal(t, tc.expected, nameForConfig(&conf))
	}
}
