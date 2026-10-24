package ecoflow

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/charger/ecoflow/ecoflowtest"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestWallbox(t *testing.T, clk clock.Clock) (*Wallbox, *ecoflowtest.Transport) {
	t.Helper()

	transport := ecoflowtest.NewTransport()
	conn := NewConnection(util.NewLogger("test"), transport, testUserID).WithClock(clk)
	wb, err := conn.Wallbox(ecoflowtest.Serial, ecoflowtest.Gateway, time.Minute)
	require.NoError(t, err)

	return wb, transport
}

// A write returns once published; it does not wait for the wallbox.
func TestWriteDoesNotWaitForTelemetry(t *testing.T) {
	wb, transport := newTestWallbox(t, clock.New())
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state3-3p")))

	start := time.Now()
	require.NoError(t, wb.SetMaxCurrent(100))
	require.NoError(t, wb.StartStop(false))
	require.NoError(t, wb.SetPhaseMode(PhaseOne))
	assert.Less(t, time.Since(start), 100*time.Millisecond)
	assert.Len(t, transport.Published(), 4, "get-all and three writes")
}

// Field 21 = 0 is reported in auto as well, so it does not confirm a 3p
// write; only 241/44 or field 21 = 1 settle it.
func TestSetPhaseModeThreeNotConfirmedByField21(t *testing.T) {
	clk := clock.NewMock()
	wb, transport := newTestWallbox(t, clk)
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state2-1p")))

	require.NoError(t, wb.SetPhaseMode(PhaseThree))
	clk.Add(settingsSettle)

	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state2-3p-fixed")))
	assert.True(t, wb.phases.open, "field 21 = 0 is no confirmation")

	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Settings("phase3p")))
	assert.False(t, wb.phases.open, "confirmed by 241/44")
}

func TestSetPhaseModeContradictedByField21(t *testing.T) {
	clk := clock.NewMock()
	wb, transport := newTestWallbox(t, clk)
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state2-1p")))

	require.NoError(t, wb.SetPhaseMode(PhaseThree))

	// right after the write the old mode is still reported
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state2-1p")))
	assert.True(t, wb.phases.open)

	clk.Add(settingsSettle)
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state2-1p")))
	assert.False(t, wb.phases.open, "still 1p after the settle time")
}

func TestWriteWithoutGateway(t *testing.T) {
	conn := NewConnection(util.NewLogger("test"), ecoflowtest.NewTransport(), testUserID)
	wb, err := conn.Wallbox(ecoflowtest.Serial, "", time.Minute)
	require.NoError(t, err)

	assert.Error(t, wb.SetMaxCurrent(100))
}
