package charger

import (
	"bytes"
	"strings"
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
		assert.Error(t, err)
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

// answerWrites makes the fake wallbox report telemetry (heartbeats or 241/44
// settings reports) after every write, as the real one does once it executed
// a command.
func answerWrites(transport *ecoflowtest.Transport, heartbeats ...[]byte) {
	transport.OnPublish(func(m ecoflowtest.Message) {
		if !strings.HasSuffix(m.Topic, "/thing/property/set") {
			return
		}
		for _, hb := range heartbeats {
			_ = transport.Feed(powerPulseTelemetry, hb)
		}
	})
}

// writes returns the publishes after the initial get-all.
func writes(transport *ecoflowtest.Transport) []ecoflowtest.Message {
	return transport.Published()[1:]
}

func TestEcoflowPowerPulseStop(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))
	answerWrites(transport, ecoflowtest.Heartbeat("state6-3p"))

	require.NoError(t, wb.Enable(false))

	// seq 1 went to the get-all
	assert.Equal(t, []ecoflowtest.Message{ecoflowtest.Write("stop", 2)}, writes(transport))
}

func TestEcoflowPowerPulseStopThenStart(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))

	answerWrites(transport, ecoflowtest.Heartbeat("state6-3p"))
	require.NoError(t, wb.Enable(false))

	answerWrites(transport, ecoflowtest.Heartbeat("state3-3p"))
	require.NoError(t, wb.Enable(true))

	assert.Equal(t, []ecoflowtest.Message{
		ecoflowtest.Write("stop", 2),
		ecoflowtest.Write("start", 3),
	}, writes(transport))

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)
}

func TestEcoflowPowerPulseEnableReadback(t *testing.T) {
	tests := []struct {
		name    string
		before  string // heartbeat before the write
		enable  bool
		answers []string // heartbeats after the write
		err     error
	}{
		{"stop confirmed", "state3-3p", false, []string{"state6-3p"}, nil},
		{"start confirmed", "state6-3p", true, []string{"state3-3p"}, nil},
		{"start confirmed by paused vehicle", "state6-3p", true, []string{"state5-3p"}, nil},
		{"stale heartbeat, then confirmed", "state3-3p", false, []string{"state3-3p", "state6-3p"}, nil},
		{"stop not confirmed", "state3-3p", false, nil, api.ErrTimeout},
		{"start not confirmed", "state6-3p", true, nil, api.ErrTimeout},
		{"only stale heartbeat", "state6-3p", true, []string{"state6-3p"}, api.ErrTimeout},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wb, transport := newTestPowerPulse(t, time.Minute)
			wb.startTimeout = 50 * time.Millisecond
			wb.stopTimeout = 50 * time.Millisecond

			require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat(tc.before)))

			var answers [][]byte
			for _, name := range tc.answers {
				answers = append(answers, ecoflowtest.Heartbeat(name))
			}
			answerWrites(transport, answers...)

			err := wb.Enable(tc.enable)
			if tc.err == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.err)
			}
		})
	}
}

// Start and stop have their own readback windows: on hardware a start took
// up to 13 s to show, a stop under 2 s.
func TestEcoflowPowerPulseEnableTimeouts(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	assert.Equal(t, 60*time.Second, wb.startTimeout)
	assert.Equal(t, 30*time.Second, wb.stopTimeout)

	// each command must give up after its own window, not the other one
	expectTimeoutWithin := func(enable bool) {
		t.Helper()
		start := time.Now()
		assert.ErrorIs(t, wb.Enable(enable), api.ErrTimeout)
		assert.Less(t, time.Since(start), time.Second)
	}

	wb.startTimeout = 3 * time.Second
	wb.stopTimeout = 10 * time.Millisecond
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))
	expectTimeoutWithin(false)

	wb.startTimeout = 10 * time.Millisecond
	wb.stopTimeout = 3 * time.Second
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))
	expectTimeoutWithin(true)
}

// A wallbox already in the requested state gets no write, so no confirmation is awaited.
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
			wb.startTimeout = 200 * time.Millisecond
			wb.stopTimeout = 200 * time.Millisecond
			require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat(tc.before)))

			require.NoError(t, wb.Enable(tc.enable))
			assert.Empty(t, writes(transport))
		})
	}
}

// Without a vehicle the wallbox cannot start, so the start is sent but not awaited.
func TestEcoflowPowerPulseStartWithoutVehicle(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	wb.startTimeout = 200 * time.Millisecond
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state1-3p")))

	require.NoError(t, wb.Enable(true))
	assert.Equal(t, []ecoflowtest.Message{ecoflowtest.Write("start", 2)}, writes(transport))
}

// The wallbox sends no set_reply; a confirming heartbeat alone suffices and
// the charger subscribes to nothing but the telemetry topic.
func TestEcoflowPowerPulseNoSetReply(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))
	answerWrites(transport, ecoflowtest.Heartbeat("state6-3p"))

	require.NoError(t, wb.Enable(false))
	assert.Equal(t, map[string]byte{powerPulseTelemetry: 0}, transport.Subscriptions())
}

func TestEcoflowPowerPulseMaxCurrent(t *testing.T) {
	tests := []struct {
		name    string
		current int64
		before  string // heartbeat before the write
		answer  []byte // settings report after the write
	}{
		// 15 -> 10 A while charging, stage 4a
		{"max-current-10", 10, "state3-3p-15a", ecoflowtest.Settings("max10")},
		// 10 -> 16 A while charging, stage 4b-7
		{"max-current-16", 16, "state3-1p", ecoflowtest.Settings("max16")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wb, transport := newTestPowerPulse(t, time.Minute)
			wb.maxCurrentTimeout = time.Second
			require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat(tc.before)))
			answerWrites(transport, tc.answer)

			require.NoError(t, wb.MaxCurrent(tc.current))
			assert.Equal(t, []ecoflowtest.Message{ecoflowtest.Write(tc.name, 2)}, writes(transport))
		})
	}
}

func TestEcoflowPowerPulseMaxCurrentOutOfRange(t *testing.T) {
	for _, current := range []int64{0, 5, 17, 32} {
		wb, transport := newTestPowerPulse(t, time.Minute)
		wb.maxCurrentTimeout = 50 * time.Millisecond
		require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state3-3p")))
		answerWrites(transport, ecoflowtest.Settings("max16"))

		assert.Error(t, wb.MaxCurrent(current), "%d A", current)
		assert.Empty(t, writes(transport), "%d A", current)
	}
}

func TestEcoflowPowerPulseMaxCurrentReadback(t *testing.T) {
	tests := []struct {
		name    string
		current int64
		before  string   // heartbeat before the write
		answers [][]byte // telemetry after the write
		err     error
	}{
		{"241/44", 10, "state3-3p-15a", [][]byte{ecoflowtest.Settings("max10")}, nil},
		{"241/44 after stale report", 10, "state3-3p-15a", [][]byte{
			ecoflowtest.Settings("max15"), ecoflowtest.Settings("no-params"), ecoflowtest.Settings("max10"),
		}, nil},
		{"heartbeat only, no 241/44", 16, "state3-1p", [][]byte{ecoflowtest.Heartbeat("state3-3p")}, nil},
		{"heartbeat only, 10 A", 10, "state3-3p", [][]byte{ecoflowtest.Heartbeat("state3-1p")}, nil},
		{"no readback", 10, "state3-3p-15a", nil, api.ErrTimeout},
		{"only stale telemetry", 10, "state3-3p-15a", [][]byte{
			ecoflowtest.Settings("max15"), ecoflowtest.Settings("no-params"), ecoflowtest.Heartbeat("state3-3p-15a"),
		}, api.ErrTimeout},
		// a matching heartbeat from before the write is no confirmation
		{"already set, no telemetry after write", 10, "state3-1p", nil, api.ErrTimeout},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wb, transport := newTestPowerPulse(t, time.Minute)
			wb.maxCurrentTimeout = 50 * time.Millisecond

			require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat(tc.before)))
			answerWrites(transport, tc.answers...)

			err := wb.MaxCurrent(tc.current)
			if tc.err == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.err)
			}
		})
	}
}

// Without 241/44 the next heartbeat may take up to 60 s.
func TestEcoflowPowerPulseMaxCurrentTimeout(t *testing.T) {
	wb, _ := newTestPowerPulse(t, time.Minute)
	assert.Equal(t, 70*time.Second, wb.maxCurrentTimeout)
}

func TestEcoflowPowerPulsePhases1p(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Minute)
	wb.phasesTimeout = time.Second
	// stopped at 3p, stage 4b-2
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))
	answerWrites(transport, ecoflowtest.Settings("phase1p"))

	require.NoError(t, wb.Phases1p3p(1))
	assert.Equal(t, []ecoflowtest.Message{ecoflowtest.Write("phase-1p", 2)}, writes(transport))
}

// No 3p write was captured. The phase-auto capture differs from it only in
// settings field 5, so the expected frame is that capture with 0 replaced by 2.
func TestEcoflowPowerPulsePhases3p(t *testing.T) {
	want := ecoflowtest.Write("phase-auto", 2)
	settingsAuto := []byte{0x22, 0x02, 0x28, 0x00} // field 4 { field 5 = 0 }
	require.Equal(t, 1, bytes.Count(want.Payload, settingsAuto))
	want.Payload = bytes.Replace(want.Payload, settingsAuto, []byte{0x22, 0x02, 0x28, 0x02}, 1)

	wb, transport := newTestPowerPulse(t, time.Minute)
	wb.phasesTimeout = time.Second
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state2-1p")))
	// field 21 = 0: all three phases in effect
	answerWrites(transport, ecoflowtest.Heartbeat("state2-3p"))

	require.NoError(t, wb.Phases1p3p(3))
	assert.Equal(t, []ecoflowtest.Message{want}, writes(transport))
}

// Switching under load is refused; nothing is sent.
func TestEcoflowPowerPulsePhasesRefusedWhileCharging(t *testing.T) {
	for _, phases := range []int{1, 3} {
		for _, before := range []string{"state3-3p", "state3-1p"} {
			wb, transport := newTestPowerPulse(t, time.Minute)
			wb.phasesTimeout = 50 * time.Millisecond
			require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat(before)))
			answerWrites(transport, ecoflowtest.Settings("phase1p"), ecoflowtest.Heartbeat("state2-3p"))

			assert.Error(t, wb.Phases1p3p(phases), "%dp from %s", phases, before)
			assert.Empty(t, writes(transport), "%dp from %s", phases, before)
		}
	}
}

func TestEcoflowPowerPulsePhasesInvalid(t *testing.T) {
	for _, phases := range []int{0, 2, 4} {
		wb, transport := newTestPowerPulse(t, time.Minute)
		wb.phasesTimeout = 50 * time.Millisecond
		require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))
		answerWrites(transport, ecoflowtest.Settings("phase1p"), ecoflowtest.Heartbeat("state2-1p"))

		assert.Error(t, wb.Phases1p3p(phases), "%d phases", phases)
		assert.Empty(t, writes(transport), "%d phases", phases)
	}
}

// Without telemetry the charger cannot tell whether it is charging.
func TestEcoflowPowerPulsePhasesWithoutTelemetry(t *testing.T) {
	wb, transport := newTestPowerPulse(t, time.Millisecond)
	wb.phasesTimeout = 50 * time.Millisecond

	assert.ErrorIs(t, wb.Phases1p3p(1), api.ErrOutdated)
	assert.Empty(t, writes(transport))
}

func TestEcoflowPowerPulsePhasesReadback(t *testing.T) {
	tests := []struct {
		name    string
		phases  int
		before  string   // heartbeat before the write
		answers [][]byte // telemetry after the write
		err     error
	}{
		// right after the write the wallbox still reports the old mode, stage 4b-2
		{"1p: 241/44 after stale report", 1, "state6-3p", [][]byte{
			ecoflowtest.Settings("max10"), ecoflowtest.Settings("phase1p"),
		}, nil},
		{"1p: heartbeat only", 1, "state6-3p", [][]byte{ecoflowtest.Heartbeat("state2-1p")}, nil},
		{"3p: heartbeat only", 3, "state2-1p", [][]byte{ecoflowtest.Heartbeat("state6-3p")}, nil},
		{"1p: only stale telemetry", 1, "state6-3p", [][]byte{
			ecoflowtest.Settings("phaseauto"), ecoflowtest.Heartbeat("state6-3p"),
		}, api.ErrTimeout},
		// auto is not three phases, and 1p is the old mode
		{"3p: 241/44 reports auto or 1p", 3, "state2-1p", [][]byte{
			ecoflowtest.Settings("phaseauto"), ecoflowtest.Settings("phase1p"),
		}, api.ErrTimeout},
		// field 21 absent must not read as 0 = three phases
		{"3p: heartbeat without phase mode", 3, "state2-1p", [][]byte{ecoflowtest.HeartbeatWithState(2)}, api.ErrTimeout},
		{"no readback", 1, "state6-3p", nil, api.ErrTimeout},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wb, transport := newTestPowerPulse(t, time.Minute)
			wb.phasesTimeout = 50 * time.Millisecond

			require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat(tc.before)))
			answerWrites(transport, tc.answers...)

			err := wb.Phases1p3p(tc.phases)
			if tc.err == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.err)
			}
		})
	}
}

// Without 241/44 the next heartbeat may take up to 60 s.
func TestEcoflowPowerPulsePhasesTimeout(t *testing.T) {
	wb, _ := newTestPowerPulse(t, time.Minute)
	assert.Equal(t, 70*time.Second, wb.phasesTimeout)
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wb, transport := newTestPowerPulse(t, time.Minute)
			wb.phasesTimeout = 200 * time.Millisecond
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
	wb.phasesTimeout = 50 * time.Millisecond
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Heartbeat("state6-3p")))
	require.NoError(t, transport.Feed(powerPulseTelemetry, ecoflowtest.Settings("phaseauto")))

	assert.ErrorIs(t, wb.Phases1p3p(3), api.ErrTimeout)
	assert.Len(t, writes(transport), 1)
}
