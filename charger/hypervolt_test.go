package charger

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHypervolt() *Hypervolt {
	return &Hypervolt{
		log:     util.NewLogger("test"),
		pending: make(map[string]chan error),
		ready:   make(chan struct{}),
		initErr: make(chan error, 1),
		timeout: time.Second,
	}
}

func isReady(wb *Hypervolt) bool {
	select {
	case <-wb.ready:
		return true
	default:
		return false
	}
}

func TestHypervoltSnapshotReply(t *testing.T) {
	wb := newTestHypervolt()
	done := make(chan error, 1)
	wb.pending["0"] = done

	require.NoError(t, wb.handle([]byte(
		`{"jsonrpc":"2.0","id":"0","result":[{"brightness":0.25},{"lock_state":"unlocked"},{"release_state":"default"},{"max_current":32000},{"solar_mode":"boost"}]}`,
	)))

	assert.NoError(t, <-done)
	assert.Empty(t, wb.pending)

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	cur, err := wb.GetMaxCurrent()
	require.NoError(t, err)
	assert.Equal(t, 32.0, cur)

	// pushed update releases the charger
	require.NoError(t, wb.handle([]byte(`{"method":"sync.apply","params":[{"release_state":"released"}]}`)))

	enabled, err = wb.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestHypervoltErrorReplyReachesCaller(t *testing.T) {
	wb := newTestHypervolt()
	done := make(chan error, 1)
	wb.pending["1"] = done

	require.NoError(t, wb.handle([]byte(
		`{"jsonrpc":"2.0","id":"1","error":{"code":409,"message":"Concurrent modifications invalidated this request","data":null}}`,
	)))

	err := <-done
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Concurrent modifications")
	assert.Empty(t, wb.pending, "pending entry must be removed")
}

func TestHypervoltLoginRejected(t *testing.T) {
	wb := newTestHypervolt()
	done := make(chan error, 1)
	wb.pending["1"] = done

	require.NoError(t, wb.handle([]byte(`{"jsonrpc":"2.0","id":"1","result":{"authenticated":false}}`)))
	assert.Error(t, <-done)

	done = make(chan error, 1)
	wb.pending["2"] = done
	require.NoError(t, wb.handle([]byte(`{"jsonrpc":"2.0","id":"2","error":{"code":-32601,"message":"login not allowed","data":null}}`)))
	assert.Error(t, <-done)
}

func TestHypervoltCommandNotConnected(t *testing.T) {
	wb := newTestHypervolt()
	assert.Error(t, wb.Enable(true), "commands must not be sent before login")
	assert.Error(t, wb.MaxCurrent(16))
}

func TestHypervoltReadyWaitsForSession(t *testing.T) {
	wb := newTestHypervolt()

	require.NoError(t, wb.handle([]byte(`{"jsonrpc":"2.0","method":"sync.apply","params":[{"release_state":"default"}]}`)))
	assert.False(t, isReady(wb), "must wait for the initial session before signalling ready")

	require.NoError(t, wb.handle([]byte(`{"jsonrpc":"2.0","method":"get.session","params":{"charging":true,"watt_hours":3119,"ev_power":7192}}`)))
	assert.True(t, isReady(wb))

	energy, err := wb.ChargedEnergy()
	require.NoError(t, err)
	assert.Equal(t, 3.119, energy, "startup baseline includes current session energy")
}

func TestHypervoltReadyAfterSessionThenSync(t *testing.T) {
	wb := newTestHypervolt()

	require.NoError(t, wb.handle([]byte(`{"jsonrpc":"2.0","method":"get.session","params":{"charging":false,"watt_hours":0,"ev_power":0}}`)))
	assert.False(t, isReady(wb))

	require.NoError(t, wb.handle([]byte(`{"jsonrpc":"2.0","method":"sync.apply","params":[{"release_state":"default"}]}`)))
	assert.True(t, isReady(wb))
}

func TestHypervoltPilotStatus(t *testing.T) {
	wb := newTestHypervolt()

	for _, tc := range []struct {
		pilot  string
		status api.ChargeStatus
	}{
		{"A", api.StatusA},
		{"B", api.StatusB},
		{"C", api.StatusC},
	} {
		require.NoError(t, wb.handle([]byte(
			`{"method":"get.pilot_status","params":{"pilot_status":"`+tc.pilot+`"}}`,
		)))

		status, err := wb.Status()
		require.NoError(t, err)
		assert.Equal(t, tc.status, status, tc.pilot)
	}
}

func TestHypervoltIdleNotStale(t *testing.T) {
	wb := newTestHypervolt()
	wb.pilot = "A"
	released := false
	wb.released = &released

	// connected but idle: no updates for a long time is fine
	status, err := wb.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusA, status)

	// disconnected briefly: still fine
	wb.disconnected = time.Now()
	_, err = wb.Status()
	assert.NoError(t, err)

	// disconnected for too long: stale
	wb.disconnected = time.Now().Add(-2 * hypervoltStaleness)
	_, err = wb.Status()
	assert.ErrorIs(t, err, api.ErrTimeout)
	_, err = wb.Enabled()
	assert.ErrorIs(t, err, api.ErrTimeout)
}

func TestHypervoltUnknownBeforeData(t *testing.T) {
	wb := newTestHypervolt()

	_, err := wb.Enabled()
	assert.Error(t, err, "enabled must not be guessed before data arrives")

	_, err = wb.Status()
	assert.Error(t, err)
}

// TestHypervoltCapturedSession replays messages captured from a Home 3 Pro while charging
func TestHypervoltCapturedSession(t *testing.T) {
	wb := newTestHypervolt()
	login := make(chan error, 1)
	wb.pending["1"] = login

	for _, msg := range []string{
		`{"jsonrpc":"2.0","id":"1","result":{"authenticated":true}}`,
		`{"jsonrpc":"2.0","method":"sync.apply","params":[{"brightness":0.5},{"lock_state":"unlocked"},{"release_state":"default"},{"max_current":32000},{"ct_flags":0},{"solar_mode":"boost"},{"features":["mms","home-battery-drain-prevention","enhanced-energy"]},{"random_start":false},{"effect_name":"none"}]}`,
		`{"jsonrpc":"2.0","method":"get.connection_state","params":{"connection_state":"online"}}`,
		`{"jsonrpc":"2.0","method":"get.pilot_status","params":{"pilot_status":"C","charging_started_millis":1795120560825}}`,
		`{"jsonrpc":"2.0","method":"get.composite_pilot_status","params":{"composite_pilot_status":"C2"}}`,
		`{"jsonrpc":"2.0","method":"get.thermal_ilimit","params":{"thermal_ilimit":31.5}}`,
		`{"jsonrpc":"2.0","method":"get.session","params":{"charging":true,"true_milli_amps":30500,"voltage":237,"watt_hours":3119,"carbon_saved_grams":1363,"ct_current":36400,"ev_power":7192,"grid_power":8631,"house_power":1439,"generation_power":0,"energy_flow":["grid_to_ev","grid_to_house"]}}`,
		`{"jsonrpc":"2.0","method":"flex.status","params":{"mpan":"0","active_energy_provider":"Octopus","registration_state":"unreg","is_flex_eligible":false}}`,
	} {
		require.NoError(t, wb.handle([]byte(msg)))
	}

	assert.NoError(t, <-login)
	assert.True(t, isReady(wb))

	status, err := wb.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	power, err := wb.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 7192.0, power)

	energy, err := wb.ChargedEnergy()
	require.NoError(t, err)
	assert.Equal(t, 3.119, energy)

	cur, err := wb.GetMaxCurrent()
	require.NoError(t, err)
	assert.Equal(t, 32.0, cur)
}
