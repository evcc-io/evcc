package cmd

import (
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

func TestParseLogLevels(t *testing.T) {
	t.Cleanup(func() {
		viper.Set("log", "info")
		util.LogLevel("info", nil)
	})

	viper.Set("log", "debug,easee:trace")
	level, levels := parseLogLevels()
	assert.Equal(t, "debug", level)
	assert.Equal(t, map[string]string{"easee": "trace"}, levels)
}
