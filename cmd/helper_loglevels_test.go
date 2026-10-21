package cmd

import (
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

func TestParseLogLevels(t *testing.T) {
	t.Cleanup(func() {
		util.LogLevelDefaults("info", nil)
	})

	viper.Set("log", "debug,easee:trace")
	parseLogLevels()

	level, areas := util.ActiveLogLevel()
	assert.Equal(t, "DEBUG", level)
	assert.Equal(t, map[string]string{"easee": "TRACE"}, areas)

	viper.Set("log", "easee:trace")
	parseLogLevels()

	level, _ = util.ActiveLogLevel()
	assert.Equal(t, "INFO", level)
}
