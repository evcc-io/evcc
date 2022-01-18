package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

func TestTimer(t *testing.T) {
	at := NewActiveTimer(util.NewLogger("foo"))

	clck := clock.NewMock()
	at.clck = clck

	at.Start()
	clck.Add(time.Minute)
	at.Reset()
	clck.Add(time.Minute)

	require.Equal(t, time.Duration(0), at.lastduration)

	// continue
	at.Start()
	clck.Add(2 * time.Minute)
	at.Stop()

	require.Equal(t, time.Duration(2*time.Minute), at.lastduration)

	// continue - should do nothing as the timer was started allready
	at.Start()
	clck.Add(1 * time.Minute)
	at.Stop()

	require.Equal(t, time.Duration(2*time.Minute), at.lastduration)
}
