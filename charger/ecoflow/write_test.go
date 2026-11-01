package ecoflow

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
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
	require.NoError(t, wb.SetCurrent(100))
	require.NoError(t, wb.StartStop(false))
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state6-3p")))
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

// Start, stop and phase mode go through the gateway; the current setpoint
// goes to the wallbox itself.
func TestWriteWithoutGateway(t *testing.T) {
	transport := ecoflowtest.NewTransport()
	conn := NewConnection(util.NewLogger("test"), transport, testUserID)
	wb, err := conn.Wallbox(ecoflowtest.Serial, "", time.Minute)
	require.NoError(t, err)

	assert.Error(t, wb.startStop(true))
	assert.Error(t, wb.writePhaseMode(PhaseOne))
	assert.NoError(t, wb.SetCurrent(100))
}

// The setpoint is a 2/81 on the topic of the wallbox, not a 241/102 through the
// gateway: lowering the maximum (241/102) under load faults the wallbox.
func TestSetCurrentWritesToWallboxTopic(t *testing.T) {
	wb, transport := newTestWallbox(t, clock.New())
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state3-3p")))

	require.NoError(t, wb.SetCurrent(130))
	msgs := transport.Published()[1:]
	require.Len(t, msgs, 1)
	assert.Equal(t, "/app/"+testUserID+"/"+ecoflowtest.Serial+"/thing/property/set", msgs[0].Topic)
	assert.Equal(t, ecoflowtest.Write("current-13", 2).Payload, msgs[0].Payload)
}

// The wallbox does not store a custom current above its maximum.
func TestSetCurrentAboveMaximum(t *testing.T) {
	tests := []struct {
		name  string
		feed  [][]byte
		raw   uint64
		error string
	}{
		{"heartbeat field 18", [][]byte{ecoflowtest.Heartbeat("state3-3p-15a")}, 160, "16.0 A above the wallbox maximum 15.0 A"},
		{"241/44 wins", [][]byte{ecoflowtest.Heartbeat("state3-3p"), ecoflowtest.Settings("max15")}, 160, "16.0 A above the wallbox maximum 15.0 A"},
		{"at the maximum", [][]byte{ecoflowtest.Heartbeat("state3-3p-15a")}, 150, ""},
		{"maximum unknown", [][]byte{ecoflowtest.HeartbeatWithState(3)}, 320, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wb, transport := newTestWallbox(t, clock.New())
			for _, p := range tc.feed {
				require.NoError(t, transport.Feed(testTelemetry, p))
			}

			err := wb.SetCurrent(tc.raw)
			if tc.error == "" {
				require.NoError(t, err)
				assert.Len(t, transport.Published(), 2)
				return
			}
			assert.ErrorContains(t, err, tc.error)
			assert.Len(t, transport.Published(), 1, "get-all only, nothing written")
		})
	}
}

// 241/44 shows the written current within about a second, but only reports
// received after the settle time count.
func TestSetCurrentConfirmedBySettings(t *testing.T) {
	clk := clock.NewMock()
	wb, transport := newTestWallbox(t, clk)
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state3-3p")))

	require.NoError(t, wb.SetCurrent(130))
	clk.Add(settingsSettle)

	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Settings("custom16")))
	assert.False(t, wb.current.open, "contradicted by 241/44")
	assert.Equal(t, uint64(130), wb.current.value)

	require.NoError(t, wb.SetCurrent(130))
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Settings("custom13")))
	assert.True(t, wb.current.open, "report within the settle time")

	clk.Add(settingsSettle)
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Settings("custom13")))
	assert.False(t, wb.current.open, "confirmed by 241/44")
}

// Heartbeat field 17 may be 0.1 A off (wrote 160, showed 159).
func TestSetCurrentConfirmedByHeartbeatWithinTolerance(t *testing.T) {
	clk := clock.NewMock()
	wb, transport := newTestWallbox(t, clk)
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state3-3p-set13")))

	require.NoError(t, wb.SetCurrent(160))
	assert.True(t, wb.current.open)

	clk.Add(time.Minute)
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state3-3p-set159")))
	assert.False(t, wb.current.open, "159 confirms 160")

	current, err := wb.Current()
	require.NoError(t, err)
	assert.Equal(t, uint64(160), current, "the written value, not 0.1 A less")
}

func TestCurrentWithoutSetpoint(t *testing.T) {
	wb, transport := newTestWallbox(t, clock.New())
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.HeartbeatWithState(3)))

	_, err := wb.Current()
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}

// A start or stop is logged once a heartbeat shows it, like current and phase writes.
func TestStartStopConfirmed(t *testing.T) {
	clk := clock.NewMock()
	wb, transport := newTestWallbox(t, clk)
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state3-3p")))

	require.NoError(t, wb.StartStop(false))
	assert.True(t, wb.release.open)

	clk.Add(stopSettle)
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state6-3p")))
	assert.False(t, wb.release.open, "confirmed by state 6")

	require.NoError(t, wb.StartStop(true))
	assert.True(t, wb.release.open)

	clk.Add(startSettle)
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state2-3p")))
	assert.True(t, wb.release.open, "state 2 is ambiguous")

	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state5-3p")))
	assert.False(t, wb.release.open, "confirmed by state 5")
}
