package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogLevelReplacesAreas(t *testing.T) {
	LogLevelDefaults("info", map[string]string{"easee": "trace"})
	t.Cleanup(func() { LogLevelDefaults("info", nil) })

	LogLevel("warn", map[string]string{"Site": "debug"})
	level, areas := ActiveLogLevel()
	assert.Equal(t, "WARN", level)
	assert.Equal(t, map[string]string{"site": "DEBUG"}, areas)

	level, areas = ResetLogLevel()
	assert.Equal(t, "INFO", level)
	assert.Equal(t, map[string]string{"easee": "TRACE"}, areas)
}
