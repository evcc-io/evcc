package util

import (
	"testing"

	"github.com/evcc-io/evcc/util/logstash"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	log := NewLogger("test")
	log.TRACE.Print("foo")

	require.Len(t, logstash.All(nil, jww.LevelTrace, 0), 1)
}
