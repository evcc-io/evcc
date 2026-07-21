package util

import (
	"log/slog"
	"testing"

	"github.com/evcc-io/evcc/util/logstash"
	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	log := NewLogger("test")
	log.TRACE.Print("foo")

	require.Len(t, logstash.All(nil, logstash.LevelTrace, 0), 1)
}

func TestLogFormat(t *testing.T) {
	log := NewLogger("fmt").With("component", "loadpoint", "title", "Garage 1")
	log.WARN.Println("hello")

	all := logstash.All([]string{"fmt"}, logstash.LevelTrace, 0)
	require.Len(t, all, 1)
	require.Regexp(t, `^\[fmt   \] WARN \d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} hello component=loadpoint title="Garage 1"\n$`, all[0])

	// level filter
	require.Empty(t, logstash.All([]string{"fmt"}, logstash.LevelFatal, 0))
	require.Len(t, logstash.All([]string{"fmt"}, slog.LevelWarn, 0), 1)
}
