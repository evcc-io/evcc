package charger

import (
	"bytes"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ecoflow"
	"github.com/evcc-io/evcc/charger/ecoflow/ecoflowtest"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	powerPulseUserID    = "1000000000000000001"
	powerPulseTelemetry = "/app/device/property/" + ecoflowtest.Serial
)

func newTestPowerPulse(t *testing.T, timeout time.Duration) (*EcoflowPowerPulse, *ecoflowtest.Transport) {
	t.Helper()
	return newTestPowerPulseWithClock(t, timeout, clock.New())
}

// newTestPowerPulseWithClock lets a test age the telemetry with a mock clock.
// A mock clock blocks the first read until a heartbeat arrives, so feed one first.
func newTestPowerPulseWithClock(t *testing.T, timeout time.Duration, clk clock.Clock) (*EcoflowPowerPulse, *ecoflowtest.Transport) {
	t.Helper()

	transport := ecoflowtest.NewTransport()
	conn := ecoflow.NewConnection(util.NewLogger("test"), transport, powerPulseUserID).WithClock(clk)

	wb, err := NewEcoflowPowerPulse(conn, ecoflowtest.Serial, ecoflowtest.Gateway, timeout)
	require.NoError(t, err)

	return wb, transport
}

func TestEcoflowPowerPulseRequiresAppKey(t *testing.T) {
	_, err := NewEcoflowPowerPulseFromConfig(t.Context(), map[string]any{
		"user": "user@example.com", "password": "secret", "serial": ecoflowtest.Serial,
	})
	assert.ErrorContains(t, err, "appkey")
}

func TestEcoflowPowerPulseStatus(t *testing.T) {
	tests := []struct {
		fixture string
		status  api.ChargeStatus
		enabled bool
	}{
		{"state1-3p", api.StatusA, false},
		{"state2-3p", api.StatusB, false},
		{"state2-1p", api.StatusB, false},
		{"state3-3p", api.StatusC, true},
		{"state3-1p", api.StatusC, true},
		{"state5-3p", api.StatusB, true},
		{"state6-3p", api.StatusB, false},
	}

	for _, tc := range tests {
		t.Run(tc.fixture, func(t *testing.T) {
			wb, transport := newTestPowerPulse(t, time.Minute)
			require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat(tc.fixture)))

			status, err := wb.Status()
			require.NoError(t, err)
			assert.Equal(t, tc.status, status)

			enabled, err := wb.Enabled()
			require.NoError(t, err)
			assert.Equal(t, tc.enabled, enabled)
		})
	}
}

// States 4 and 9 come from the reference implementation and were not observed
// on hardware, so they are fed as synthetic heartbeats.
func TestEcoflowPowerPulseUnobservedStates(t *testing.T) {
	{
		wb, transport := newTestPowerPulse(t, time.Minute)
		require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.HeartbeatWithState(4)))

		status, err := wb.Status()
		require.NoError(t, err)
		assert.Equal(t, api.StatusB, status)

		enabled, err := wb.Enabled()
		require.NoError(t, err)
		assert.True(t, enabled)
	}
	{
		wb, transport := newTestPowerPulse(t, time.Minute)
		require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.HeartbeatWithState(9)))

		_, err := wb.Status()
		assert.ErrorContains(t, err, "fault")
	}
}

// The latest heartbeat wins.
func TestEcoflowPowerPulseStatusFollowsHeartbeats(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)

	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))

	status, err := wb.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusB, status)
}

func TestEcoflowPowerPulseWithoutTelemetry(t *testing.T) {
	wb, _ := newTestPowerPulse(t, time.Millisecond)

	_, err := wb.Status()
	assert.ErrorIs(t, err, api.ErrOutdated)
}

// On start the charger listens to the wallbox telemetry and asks for a
// snapshot, because the wallbox otherwise only reports once a minute.
func TestEcoflowPowerPulseRequestsSnapshot(t *testing.T) {
	_, transport := newTestPowerPulse(t, time.Minute)

	assert.Equal(t, map[string]byte{powerPulseTelemetry: 0}, transport.Subscriptions())

	published := transport.Published()
	require.Len(t, published, 1)
	assert.Equal(t, "/app/"+powerPulseUserID+"/"+ecoflowtest.Serial+"/thing/property/get", published[0].Topic)
	assert.Equal(t, byte(1), published[0].Qos)

	headers, err := ecoflow.ParseMessage(published[0].Payload)
	require.NoError(t, err)
	require.Len(t, headers, 1)
	_, _, isCommand := headers[0].Command()
	assert.False(t, isCommand, "get-all carries no command")
}

func TestEcoflowPowerPulseMeter(t *testing.T) {
	tests := []struct {
		fixture  string
		power    float64
		currents [3]float64
		voltages [3]float64
	}{
		// three phases at 15 A, 2026-10-23 12:38
		{"state3-3p-15a", 10254.4, [3]float64{15.08, 15.10, 15.16}, [3]float64{223.80, 230.59, 226.11}},
		// single phase at 10 A, only L1 carries current
		{"state3-1p", 2217.0, [3]float64{10.04, 0, 0}, [3]float64{222.10, 232.86, 231.98}},
		// plugged in, idle
		{"state2-3p", 0, [3]float64{0, 0, 0}, [3]float64{227.18, 229.00, 233.81}},
	}

	for _, tc := range tests {
		t.Run(tc.fixture, func(t *testing.T) {
			wb, transport := newTestPowerPulse(t, time.Minute)
			require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat(tc.fixture)))

			power, err := wb.CurrentPower()
			require.NoError(t, err)
			assert.InDelta(t, tc.power, power, 0.1)

			l1, l2, l3, err := wb.Currents()
			require.NoError(t, err)
			assert.InDeltaSlice(t, tc.currents[:], []float64{l1, l2, l3}, 0.01)

			u1, u2, u3, err := wb.Voltages()
			require.NoError(t, err)
			assert.InDeltaSlice(t, tc.voltages[:], []float64{u1, u2, u3}, 0.01)
		})
	}
}

func TestEcoflowPowerPulseTotalEnergy(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))

	energy, err := wb.TotalEnergy()
	require.NoError(t, err)
	assert.InDelta(t, 3679.185, energy, 0.0001)
}

// A heartbeat without the energy counter must not reset the meter to zero.
func TestEcoflowPowerPulseTotalEnergyKeepsLastValue(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.HeartbeatWithState(3)))

	energy, err := wb.TotalEnergy()
	require.NoError(t, err)
	assert.InDelta(t, 3679.185, energy, 0.0001)
}

func TestEcoflowPowerPulseTotalEnergyUnknown(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.HeartbeatWithState(3)))

	_, err := wb.TotalEnergy()
	assert.Error(t, err)
}

// Readings older than the timeout must not be passed off as current.
func TestEcoflowPowerPulseMeterOutdated(t *testing.T) {
	clk := clock.NewMock()
	wb, transport := newTestPowerPulseWithClock(t, time.Minute, clk)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p-15a")))

	clk.Add(time.Minute + time.Second)

	_, err := wb.CurrentPower()
	assert.ErrorIs(t, err, api.ErrOutdated)

	_, err = wb.TotalEnergy()
	assert.ErrorIs(t, err, api.ErrOutdated)

	_, _, _, err = wb.Currents()
	assert.ErrorIs(t, err, api.ErrOutdated)

	_, _, _, err = wb.Voltages()
	assert.ErrorIs(t, err, api.ErrOutdated)

	// a fresh heartbeat makes the readings current again
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p-15a")))

	_, err = wb.CurrentPower()
	assert.NoError(t, err)
}

// Without charge readings in the heartbeat the phases are unknown.
func TestEcoflowPowerPulseWithoutChargeReadings(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.HeartbeatWithState(2)))

	_, _, _, err := wb.Currents()
	assert.Error(t, err)

	_, _, _, err = wb.Voltages()
	assert.Error(t, err)
}

// writes returns the publishes after the initial get-all.
func writes(transport *ecoflowtest.Transport) []ecoflowtest.Message {
	return transport.Published()[1:]
}

func TestEcoflowPowerPulseStop(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))

	require.NoError(t, wb.Enable(false))

	// seq 1 went to the get-all
	assert.Equal(t, []ecoflowtest.Message{ecoflowtest.Write("stop", 2)}, writes(transport))
}

func TestEcoflowPowerPulseStopThenStart(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))

	require.NoError(t, wb.Enable(false))
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))
	require.NoError(t, wb.Enable(true))

	assert.Equal(t, []ecoflowtest.Message{
		ecoflowtest.Write("stop", 2),
		ecoflowtest.Write("start", 3),
	}, writes(transport))

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled, "the start is in effect before the wallbox reports it")
}

// Enable publishes and returns: the loadpoint must not stall until the
// wallbox reports the new state, which may take a minute.
func TestEcoflowPowerPulseEnableDoesNotWait(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))

	start := time.Now()
	require.NoError(t, wb.Enable(true))
	assert.Less(t, time.Since(start), 100*time.Millisecond)
	assert.Len(t, writes(transport), 1)
}

// Enabled follows the last command. Telemetry overrides it only once the
// wallbox had time to act and only when it shows charging or a stop: a
// plugged, idle wallbox (state 2) may be released but not drawing.
func TestEcoflowPowerPulseEnabledFollowsCommand(t *testing.T) {
	tests := []struct {
		name    string
		before  string // heartbeat before the command
		enable  bool
		after   time.Duration // when the next heartbeat arrives
		answer  string
		enabled bool
	}{
		{"start, charging", "state6-3p", true, 10 * time.Second, "state3-3p", true},
		{"start, not drawing yet", "state6-3p", true, 44 * time.Second, "state2-3p", true},
		{"start, idle for long", "state6-3p", true, 5 * time.Minute, "state2-3p", true},
		{"start, stale stop report", "state6-3p", true, 10 * time.Second, "state6-3p", true},
		{"start, still stopped after 60 s", "state6-3p", true, 60 * time.Second, "state6-3p", false},
		{"start, unplugged", "state6-3p", true, 5 * time.Minute, "state1-3p", true},
		{"stop, stopped", "state3-3p", false, 2 * time.Second, "state6-3p", false},
		{"stop, idle", "state3-3p", false, 2 * time.Second, "state2-3p", false},
		{"stop, stale charging report", "state3-3p", false, 2 * time.Second, "state3-3p", false},
		{"stop, still charging after 30 s", "state3-3p", false, 30 * time.Second, "state3-3p", true},
		{"stop, paused by vehicle after 30 s", "state3-3p", false, 30 * time.Second, "state5-3p", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clk := clock.NewMock()
			wb, transport := newTestPowerPulseWithClock(t, 10*time.Minute, clk)
			require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat(tc.before)))

			require.NoError(t, wb.Enable(tc.enable))
			require.Len(t, writes(transport), 1)

			enabled, err := wb.Enabled()
			require.NoError(t, err)
			assert.Equal(t, tc.enable, enabled, "right after the command")

			clk.Add(tc.after)
			require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat(tc.answer)))

			enabled, err = wb.Enabled()
			require.NoError(t, err)
			assert.Equal(t, tc.enabled, enabled)
		})
	}
}

func TestEcoflowPowerPulseEnabledOutdated(t *testing.T) {
	clk := clock.NewMock()
	wb, transport := newTestPowerPulseWithClock(t, time.Minute, clk)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))
	require.NoError(t, wb.Enable(true))

	clk.Add(time.Minute + time.Second)
	_, err := wb.Enabled()
	assert.ErrorIs(t, err, api.ErrOutdated)
}

// Without a command the state decides, and a wallbox already in the requested
// state gets no write.
func TestEcoflowPowerPulseEnableAlreadyInState(t *testing.T) {
	tests := []struct {
		before string
		enable bool
	}{
		{"state6-3p", false},
		{"state2-3p", false},
		{"state1-3p", false},
		{"state3-3p", true},
		{"state5-3p", true},
	}

	for _, tc := range tests {
		t.Run(tc.before, func(t *testing.T) {
			wb, transport := newTestPowerPulse(t, time.Minute)
			require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat(tc.before)))

			require.NoError(t, wb.Enable(tc.enable))
			assert.Empty(t, writes(transport))
		})
	}
}

// A released wallbox that does not draw (state 2) must still get the stop.
func TestEcoflowPowerPulseStopReleasedIdleWallbox(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))
	require.NoError(t, wb.Enable(true))
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state2-3p")))

	require.NoError(t, wb.Enable(false))
	assert.Equal(t, []ecoflowtest.Message{
		ecoflowtest.Write("start", 2),
		ecoflowtest.Write("stop", 3),
	}, writes(transport))
}

// Without a vehicle the wallbox cannot start; the start is sent all the same.
func TestEcoflowPowerPulseStartWithoutVehicle(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state1-3p")))

	require.NoError(t, wb.Enable(true))
	assert.Equal(t, []ecoflowtest.Message{ecoflowtest.Write("start", 2)}, writes(transport))
}

// The wallbox sends no set_reply; the charger subscribes to nothing but the
// telemetry topic.
func TestEcoflowPowerPulseNoSetReply(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))

	require.NoError(t, wb.Enable(false))
	assert.Equal(t, map[string]byte{powerPulseTelemetry: 0}, transport.Subscriptions())
}

// The setpoint is written as 2/81 to the wallbox; the maximum is left alone.
func TestEcoflowPowerPulseMaxCurrent(t *testing.T) {
	tests := []struct {
		name    string
		current int64
		before  string // heartbeat before the write
	}{
		// 16 -> 13 A while charging, the change that faulted via the maximum
		{"current-13", 13, "state3-3p"},
		// 13 -> 16 A while charging
		{"current-16", 16, "state3-3p-set13"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wb, transport := newTestPowerPulse(t, time.Minute)
			require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat(tc.before)))

			start := time.Now()
			require.NoError(t, wb.MaxCurrent(tc.current))
			assert.Less(t, time.Since(start), 100*time.Millisecond, "no wait for the readback")
			assert.Equal(t, []ecoflowtest.Message{ecoflowtest.Write(tc.name, 2)}, writes(transport))
		})
	}
}

// 6-32 A as 241/44 accepts it; the wallbox applies its own lower limit.
func TestEcoflowPowerPulseMaxCurrentRange(t *testing.T) {
	for current, valid := range map[int64]bool{0: false, 5: false, 6: true, 16: true, 33: false} {
		wb, transport := newTestPowerPulse(t, time.Minute)
		require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))

		err := wb.MaxCurrent(current)
		if valid {
			assert.NoError(t, err, "%d A", current)
			assert.Len(t, writes(transport), 1, "%d A", current)
		} else {
			assert.Error(t, err, "%d A", current)
			assert.Empty(t, writes(transport), "%d A", current)
		}
	}
}

// A current above the maximum of the wallbox is refused, not written.
func TestEcoflowPowerPulseMaxCurrentAboveMaximum(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))

	err := wb.MaxCurrent(20)
	assert.ErrorContains(t, err, "20.0 A")
	assert.ErrorContains(t, err, "16.0 A")
	assert.Empty(t, writes(transport))
}

// The maximum of the wallbox caps the loadpoint, so it never asks for a
// setpoint the wallbox refuses. No minimum, the loadpoint keeps its own.
func TestEcoflowPowerPulseGetMinMaxCurrent(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)

	_, _, err := wb.GetMinMaxCurrent()
	assert.ErrorIs(t, err, api.ErrNotAvailable, "no telemetry yet")

	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))
	lo, hi, err := wb.GetMinMaxCurrent()
	require.NoError(t, err)
	assert.Equal(t, 0.0, lo)
	assert.Equal(t, 16.0, hi)
}

// GetMaxCurrent reports the written setpoint until telemetry can show the
// write, then what the wallbox reports, so the loadpoint sees a dropped write.
func TestEcoflowPowerPulseGetMaxCurrent(t *testing.T) {
	tests := []struct {
		name    string
		write   int64         // 0: no write
		after   time.Duration // from the write to the telemetry
		answers [][]byte
		current float64
	}{
		{"heartbeat field 17", 0, 0, [][]byte{ecoflowtest.Heartbeat("state3-1p")}, 10},
		{"241/44 custom wins", 0, 0, [][]byte{ecoflowtest.Heartbeat("state3-3p"), ecoflowtest.Settings("custom13")}, 13},
		{"241/44 without custom mode is ignored", 0, 0, [][]byte{ecoflowtest.Settings("max10")}, 16},
		{"written, no telemetry yet", 10, 0, nil, 10},
		{"written, stale 241/44", 13, time.Second, [][]byte{ecoflowtest.Settings("custom16")}, 13},
		{"written, stale heartbeat", 13, time.Second, [][]byte{ecoflowtest.Heartbeat("state3-3p")}, 13},
		{"written, 241/44 confirms", 13, 10 * time.Second, [][]byte{ecoflowtest.Settings("custom13")}, 13},
		{"written, 241/44 contradicts", 13, 10 * time.Second, [][]byte{ecoflowtest.Settings("custom16")}, 16},
		{"written, next heartbeat confirms", 13, time.Minute, [][]byte{ecoflowtest.Heartbeat("state3-3p-set13")}, 13},
		{"written, next heartbeat 0.1 A below", 16, time.Minute, [][]byte{ecoflowtest.Heartbeat("state3-3p-set159")}, 16},
		{"written, next heartbeat contradicts", 10, time.Minute, [][]byte{ecoflowtest.Heartbeat("state3-3p")}, 16},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clk := clock.NewMock()
			wb, transport := newTestPowerPulseWithClock(t, 10*time.Minute, clk)
			require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))

			if tc.write > 0 {
				require.NoError(t, wb.MaxCurrent(tc.write))
			}

			clk.Add(tc.after)
			for _, p := range tc.answers {
				require.NoError(t, transport.Feed(powerPulseTelemetry, p))
			}

			current, err := wb.GetMaxCurrent()
			require.NoError(t, err)
			assert.Equal(t, tc.current, current)
		})
	}
}

func TestEcoflowPowerPulseGetMaxCurrentUnknown(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.HeartbeatWithState(3)))

	_, err := wb.GetMaxCurrent()
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}

func TestEcoflowPowerPulsePhases1p(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	// stopped at 3p, stage 4b-2
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))

	require.NoError(t, wb.Phases1p3p(1))
	assert.Equal(t, []ecoflowtest.Message{ecoflowtest.Write("phase-1p", 2)}, writes(transport))
}

// The phase-auto capture differs from a 3p write only in settings field 5,
// so the expected frame is that capture with 0 replaced by 2.
func TestEcoflowPowerPulsePhases3p(t *testing.T) {
	want := ecoflowtest.Write("phase-auto", 2)
	settingsAuto := []byte{0x22, 0x02, 0x28, 0x00} // field 4 { field 5 = 0 }
	require.Equal(t, 1, bytes.Count(want.Payload, settingsAuto))
	want.Payload = bytes.Replace(want.Payload, settingsAuto, []byte{0x22, 0x02, 0x28, 0x02}, 1)

	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state2-1p")))

	require.NoError(t, wb.Phases1p3p(3))
	assert.Equal(t, []ecoflowtest.Message{want}, writes(transport))
}

// newChargingPowerPulse returns a wallbox charging on three phases, on a mock clock.
func newChargingPowerPulse(t *testing.T) (*EcoflowPowerPulse, *ecoflowtest.Transport, *clock.Mock) {
	t.Helper()
	clk := clock.NewMock()
	wb, transport := newTestPowerPulseWithClock(t, 10*time.Minute, clk)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))
	return wb, transport, clk
}

// awaitWrites waits for the phase switch running in the background to publish
// want, advancing the mock clock so that its waits can elapse.
func awaitWrites(t *testing.T, transport *ecoflowtest.Transport, clk *clock.Mock, want []ecoflowtest.Message) {
	t.Helper()
	require.Eventually(t, func() bool {
		if len(writes(transport)) >= len(want) {
			return true
		}
		clk.Add(time.Second)
		return false
	}, 5*time.Second, time.Millisecond)
	assert.Equal(t, want, writes(transport))
}

func assertEnabled(t *testing.T, wb *EcoflowPowerPulse, want bool, msg string) {
	t.Helper()
	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.Equal(t, want, enabled, msg)
}

// evcc switches phases while charging. The wallbox stops first, gets the phase
// mode once it reports the stop and is started again. The call returns after
// the stop; Enabled reports the restart that is to come throughout, so the
// loadpoint does not enable the wallbox before the phase mode is written.
func TestEcoflowPowerPulsePhasesWhileCharging(t *testing.T) {
	wb, transport, clk := newChargingPowerPulse(t)

	require.NoError(t, wb.Phases1p3p(1))
	assert.Equal(t, []ecoflowtest.Message{ecoflowtest.Write("stop", 2)}, writes(transport))
	assertEnabled(t, wb, true, "restarts after the switch")

	// a stale report from before the stop does not release the phase write
	clk.Add(time.Second)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))
	assert.Len(t, writes(transport), 1)

	// the loadpoint re-enables a disabled charger during a phase switch
	require.NoError(t, wb.Enable(true))
	assert.Len(t, writes(transport), 1, "the start waits for the phase write")

	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))
	awaitWrites(t, transport, clk, []ecoflowtest.Message{
		ecoflowtest.Write("stop", 2),
		ecoflowtest.Write("phase-1p", 3),
		ecoflowtest.Write("start", 4),
	})
	assertEnabled(t, wb, true, "started again")
}

// A stop requested during the switch cancels the restart.
func TestEcoflowPowerPulsePhasesWhileChargingDisabled(t *testing.T) {
	wb, transport, clk := newChargingPowerPulse(t)

	require.NoError(t, wb.Phases1p3p(1))
	require.NoError(t, wb.Enable(false))
	assert.Len(t, writes(transport), 1, "the stop is already sent")
	assertEnabled(t, wb, false, "no restart")

	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))
	awaitWrites(t, transport, clk, []ecoflowtest.Message{
		ecoflowtest.Write("stop", 2),
		ecoflowtest.Write("phase-1p", 3),
	})

	clk.Add(time.Minute)
	assert.Len(t, writes(transport), 2, "not started again")
	assertEnabled(t, wb, false, "stays stopped")
}

// A wallbox that evcc stopped but that still reports charging gets no second
// stop and is not started after the switch.
func TestEcoflowPowerPulsePhasesWhileStopping(t *testing.T) {
	wb, transport, clk := newChargingPowerPulse(t)
	require.NoError(t, wb.Enable(false))

	require.NoError(t, wb.Phases1p3p(1))
	assert.Equal(t, []ecoflowtest.Message{ecoflowtest.Write("stop", 2)}, writes(transport))

	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))
	awaitWrites(t, transport, clk, []ecoflowtest.Message{
		ecoflowtest.Write("stop", 2),
		ecoflowtest.Write("phase-1p", 3),
	})

	clk.Add(time.Minute)
	assert.Len(t, writes(transport), 2, "not started")
	assertEnabled(t, wb, false, "stays stopped")
}

// Without a reported stop the phase mode is not written under load; the
// wallbox is started again in the old mode.
func TestEcoflowPowerPulsePhasesWhileChargingStopNotReported(t *testing.T) {
	wb, transport, clk := newChargingPowerPulse(t)

	require.NoError(t, wb.Phases1p3p(1))
	awaitWrites(t, transport, clk, []ecoflowtest.Message{
		ecoflowtest.Write("stop", 2),
		ecoflowtest.Write("start", 3),
	})
	assertEnabled(t, wb, true, "started again")
}

// The current is written right away during the switch.
func TestEcoflowPowerPulsePhasesWhileChargingMaxCurrent(t *testing.T) {
	wb, transport, clk := newChargingPowerPulse(t)

	require.NoError(t, wb.Phases1p3p(1))
	require.NoError(t, wb.MaxCurrent(13))
	assert.Equal(t, []ecoflowtest.Message{
		ecoflowtest.Write("stop", 2),
		ecoflowtest.Write("current-13", 3),
	}, writes(transport))

	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))
	awaitWrites(t, transport, clk, []ecoflowtest.Message{
		ecoflowtest.Write("stop", 2),
		ecoflowtest.Write("current-13", 3),
		ecoflowtest.Write("phase-1p", 4),
		ecoflowtest.Write("start", 5),
	})
}

// A wallbox that does not charge only gets the phase mode.
func TestEcoflowPowerPulsePhasesNotCharging(t *testing.T) {
	for _, before := range []string{"state1-3p", "state2-3p", "state5-3p", "state6-3p"} {
		t.Run(before, func(t *testing.T) {
			clk := clock.NewMock()
			wb, transport := newTestPowerPulseWithClock(t, 10*time.Minute, clk)
			require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat(before)))

			require.NoError(t, wb.Phases1p3p(1))
			clk.Add(time.Minute)
			assert.Equal(t, []ecoflowtest.Message{ecoflowtest.Write("phase-1p", 2)}, writes(transport))
		})
	}
}

func TestEcoflowPowerPulsePhasesInvalid(t *testing.T) {
	for _, phases := range []int{0, 2, 4} {
		wb, transport := newTestPowerPulse(t, time.Minute)
		require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))

		assert.Error(t, wb.Phases1p3p(phases), "%d phases", phases)
		assert.Empty(t, writes(transport), "%d phases", phases)
	}
}

// Without telemetry the charger cannot tell whether it is charging.
func TestEcoflowPowerPulsePhasesWithoutTelemetry(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Millisecond)

	assert.ErrorIs(t, wb.Phases1p3p(1), api.ErrOutdated)
	assert.Empty(t, writes(transport))
}

// A wallbox already in the requested phase mode gets no write.
func TestEcoflowPowerPulsePhasesAlreadySet(t *testing.T) {
	tests := []struct {
		name   string
		phases int
		before [][]byte // telemetry before the call
	}{
		// field 21 = 1: one phase in effect
		{"1p by heartbeat", 1, [][]byte{ecoflowtest.Heartbeat("state2-1p")}},
		// 241/44 wins over field 21 = 0
		{"1p by 241/44", 1, [][]byte{ecoflowtest.Heartbeat("state6-3p"), ecoflowtest.Settings("phase1p")}},
		{"3p by 241/44", 3, [][]byte{ecoflowtest.Heartbeat("state6-3p"), ecoflowtest.Settings("phase3p")}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wb, transport := newTestPowerPulse(t, time.Minute)
			for _, p := range tc.before {
				require.NoError(t, transport.Feed(powerPulseTelemetry, p))
			}

			require.NoError(t, wb.Phases1p3p(tc.phases))
			assert.Empty(t, writes(transport))
		})
	}
}

// Field 21 = 0 may mean auto, so 3p is written unless 241/44 reports three phases.
func TestEcoflowPowerPulsePhases3pWrittenFromAuto(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Settings("phaseauto")))

	require.NoError(t, wb.Phases1p3p(3))
	assert.Len(t, writes(transport), 1)
}
