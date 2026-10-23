package util

import (
	"testing"

	jww "github.com/spf13/jwalterweatherman"
	"github.com/stretchr/testify/assert"
)

func TestLogLevelReplacesAreas(t *testing.T) {
	t.Cleanup(func() { LogLevel("info", nil) })

	LogLevel("info", map[string]string{"easee": "trace"})
	assert.Equal(t, jww.LevelTrace, logLevelForArea("easee"))

	LogLevel("info", map[string]string{"site": "debug"})
	assert.Equal(t, jww.LevelInfo, logLevelForArea("easee"))
	assert.Equal(t, jww.LevelDebug, logLevelForArea("site"))
}
