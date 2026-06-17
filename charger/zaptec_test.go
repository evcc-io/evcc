package charger

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/charger/zaptec"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

func TestZaptecConnectionDuration(t *testing.T) {
	clck := clock.NewMock()

	var session string
	c := &Zaptec{clock: clck}
	c.statusG = util.ResettableCached(func() (zaptec.StateResponse, error) {
		return zaptec.StateResponse{
			{StateId: zaptec.SessionIdentifier, ValueAsString: session},
		}, nil
	}, 0)

	// no session yet (disconnected)
	d, err := c.ConnectionDuration()
	require.NoError(t, err)
	require.Zero(t, d)

	// session starts
	session = "session-1"
	c.statusG.Reset()
	d, err = c.ConnectionDuration()
	require.NoError(t, err)
	require.Zero(t, d)

	// time passes within the session
	clck.Add(time.Minute)
	c.statusG.Reset()
	d, err = c.ConnectionDuration()
	require.NoError(t, err)
	require.Equal(t, time.Minute, d)

	// cable swapped to another vehicle: new session, duration drops
	session = "session-2"
	c.statusG.Reset()
	d, err = c.ConnectionDuration()
	require.NoError(t, err)
	require.Zero(t, d, "duration must drop on session change to trigger reconnect")
}
