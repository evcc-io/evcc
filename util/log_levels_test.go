package util

import (
	"strconv"
	"sync"
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

func TestLogLevelConcurrent(t *testing.T) {
	t.Cleanup(func() { LogLevel("info", nil) })

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			LogLevel("info", map[string]string{"race": "debug"})
			NewLogger("race" + strconv.Itoa(i))
		})
	}
	wg.Wait()
}
