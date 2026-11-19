package charger

import (
	"context"
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
		pending: make(map[string]string),
		ready:   make(chan struct{}),
	}
}

func TestHypervoltSnapshot(t *testing.T) {
	wb := newTestHypervolt()
	wb.pending["0"] = "sync.snapshot"

	// response to sync.snapshot, matched by id
	require.NoError(t, wb.handle(context.Background(), []byte(
		`{"jsonrpc":"2.0","id":"0","result":[{"brightness":0.25},{"lock_state":"unlocked"},{"release_state":"default"},{"max_current":32000},{"solar_mode":"boost"}]}`,
	)))

	select {
	case <-wb.ready:
	default:
		t.Fatal("ready not signalled")
	}

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	cur, err := wb.GetMaxCurrent()
	require.NoError(t, err)
	assert.Equal(t, 32.0, cur)
	assert.Empty(t, wb.pending)

	// pushed update releases the charger
	require.NoError(t, wb.handle(context.Background(), []byte(
		`{"method":"sync.apply","params":[{"release_state":"released"}]}`,
	)))

	enabled, err = wb.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestHypervoltSession(t *testing.T) {
	wb := newTestHypervolt()

	require.NoError(t, wb.handle(context.Background(), []byte(
		`{"method":"get.session","params":{"charging":true,"session":123,"watt_hours":4500,"ev_power":7200,"ct_current":0,"voltage":240}}`,
	)))

	status, err := wb.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)

	power, err := wb.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 7200.0, power)

	energy, err := wb.ChargedEnergy()
	require.NoError(t, err)
	assert.Equal(t, 4.5, energy)
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
		require.NoError(t, wb.handle(context.Background(), []byte(
			`{"method":"get.pilot_status","params":{"pilot_status":"`+tc.pilot+`"}}`,
		)))

		status, err := wb.Status()
		require.NoError(t, err)
		assert.Equal(t, tc.status, status, tc.pilot)
	}
}

func TestHypervoltStaleAndUnknown(t *testing.T) {
	wb := newTestHypervolt()

	_, err := wb.Enabled()
	assert.Error(t, err, "enabled must not be guessed before data arrives")

	wb.pilot = "B"
	wb.updated = time.Now().Add(-2 * hypervoltStaleness)

	_, err = wb.Status()
	assert.ErrorIs(t, err, api.ErrTimeout)
}

// TestHypervoltCapturedSession replays messages captured from a Home 3 Pro while charging
func TestHypervoltCapturedSession(t *testing.T) {
	wb := newTestHypervolt()
	wb.pending["1"] = "login"
	wb.pending["2"] = "sync.snapshot"

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
		// login reply triggers sync.snapshot, which fails without a connection; that is expected here
		_ = wb.handle(context.Background(), []byte(msg))
	}

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

func TestHypervoltErrorIgnored(t *testing.T) {
	wb := newTestHypervolt()

	assert.NoError(t, wb.handle(context.Background(), []byte(
		`{"jsonrpc":"2.0","id":"1","error":{"code":409,"error":"Concurrent modifications invalidated this request","data":null}}`,
	)))
}
