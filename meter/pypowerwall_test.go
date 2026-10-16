package meter

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pypowerwallServer struct {
	*httptest.Server
	responses map[string]string
	auth      string
	body      string
	status    int
}

func newPypowerwallServer(t *testing.T) *pypowerwallServer {
	t.Helper()

	s := &pypowerwallServer{
		status: http.StatusOK,
		responses: map[string]string{
			"/api/meters/aggregates": `{
				"site": {"instant_power": -15, "energy_exported": 565439, "energy_imported": 205099, "i_a_current": -2.485, "i_b_current": 2.485, "i_c_current": 0},
				"battery": {"instant_power": -869, "energy_exported": 761031, "energy_imported": 779832, "i_a_current": 0, "i_b_current": 0, "i_c_current": 0},
				"load": {"instant_power": 726, "energy_exported": 0, "energy_imported": 2299854},
				"solar": {"instant_power": 1591, "energy_exported": 2678995, "energy_imported": 0}
			}`,
			"/api/system_status/soe": `{"percentage": 45.5, "raw_percentage": 48.2}`,
			"/api/system_status":     `{"nominal_full_pack_energy": 14100, "max_charge_power": null}`,
			"/api/operation":         `{"real_mode": "self_consumption", "backup_reserve_percent": 20.0}`,
			"/freq":                  `{"ISLAND_VL1N_Main": 121, "ISLAND_VL2N_Main": 117.5, "ISLAND_VL3N_Main": null, "PW1_PINV_VSplit1": 119.6, "PW1_PINV_VSplit2": 120.6}`,
		},
	}

	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			assert.Equal(t, "/control/reserve", r.URL.Path)

			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err)

			s.auth, s.body = r.Header.Get("Authorization"), string(b)
			w.WriteHeader(s.status)
			return
		}

		res, ok := s.responses[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, res)
	}))
	t.Cleanup(s.Close)

	return s
}

func (s *pypowerwallServer) meter(t *testing.T, usage string, other map[string]any) api.Meter {
	t.Helper()

	config := map[string]any{"uri": s.URL, "usage": usage}
	for k, v := range other {
		config[k] = v
	}

	m, err := NewPyPowerwallFromConfig(config)
	require.NoError(t, err)

	return m
}

func TestPyPowerwallConfigValidation(t *testing.T) {
	_, err := NewPyPowerwallFromConfig(map[string]any{"usage": "grid"})
	assert.ErrorContains(t, err, "missing uri")

	_, err = NewPyPowerwallFromConfig(map[string]any{"uri": "localhost", "usage": "load"})
	assert.ErrorContains(t, err, "invalid usage")
}

func TestPyPowerwallGrid(t *testing.T) {
	m := newPypowerwallServer(t).meter(t, "grid", nil)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, -15.0, power)

	energy, ok := api.Cap[api.MeterEnergy](m)
	require.True(t, ok)
	imported, err := energy.TotalEnergy()
	require.NoError(t, err)
	assert.InDelta(t, 205.099, imported, 1e-6)

	returnEnergy, ok := api.Cap[api.MeterReturnEnergy](m)
	require.True(t, ok)
	exported, err := returnEnergy.ReturnEnergy()
	require.NoError(t, err)
	assert.InDelta(t, 565.439, exported, 1e-6)

	currents, ok := api.Cap[api.PhaseCurrents](m)
	require.True(t, ok)
	i1, i2, i3, err := currents.Currents()
	require.NoError(t, err)
	assert.Equal(t, []float64{-2.485, 2.485, 0}, []float64{i1, i2, i3})

	// missing third phase reads as zero
	voltages, ok := api.Cap[api.PhaseVoltages](m)
	require.True(t, ok)
	u1, u2, u3, err := voltages.Voltages()
	require.NoError(t, err)
	assert.Equal(t, []float64{121, 117.5, 0}, []float64{u1, u2, u3})

	_, ok = api.Cap[api.Battery](m)
	assert.False(t, ok)
}

// connections without device vitals provide no phase data
func TestPyPowerwallGridWithoutVitals(t *testing.T) {
	for name, res := range map[string]string{"empty": `{}`, "null": `null`} {
		t.Run(name, func(t *testing.T) {
			s := newPypowerwallServer(t)
			s.responses["/freq"] = res
			m := s.meter(t, "grid", nil)

			_, ok := api.Cap[api.PhaseCurrents](m)
			assert.False(t, ok)
			_, ok = api.Cap[api.PhaseVoltages](m)
			assert.False(t, ok)
		})
	}
}

func TestPyPowerwallPV(t *testing.T) {
	m := newPypowerwallServer(t).meter(t, "pv", nil)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 1591.0, power)

	energy, ok := api.Cap[api.MeterEnergy](m)
	require.True(t, ok)
	produced, err := energy.TotalEnergy()
	require.NoError(t, err)
	assert.InDelta(t, 2678.995, produced, 1e-6)

	_, ok = api.Cap[api.MeterReturnEnergy](m)
	assert.False(t, ok)
	_, ok = api.Cap[api.PhaseCurrents](m)
	assert.False(t, ok)
}

func TestPyPowerwallBattery(t *testing.T) {
	s := newPypowerwallServer(t)
	m := s.meter(t, "battery", nil)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, -869.0, power)

	energy, ok := api.Cap[api.MeterEnergy](m)
	require.True(t, ok)
	charged, err := energy.TotalEnergy()
	require.NoError(t, err)
	assert.InDelta(t, 779.832, charged, 1e-6)

	returnEnergy, ok := api.Cap[api.MeterReturnEnergy](m)
	require.True(t, ok)
	discharged, err := returnEnergy.ReturnEnergy()
	require.NoError(t, err)
	assert.InDelta(t, 761.031, discharged, 1e-6)

	battery, ok := api.Cap[api.Battery](m)
	require.True(t, ok)
	soc, err := battery.Soc()
	require.NoError(t, err)
	assert.Equal(t, 45.5, soc)

	capacity, ok := api.Cap[api.BatteryCapacity](m)
	require.True(t, ok)
	assert.InDelta(t, 14.1, capacity.Capacity(), 1e-6)

	limiter, ok := api.Cap[api.BatterySocLimiter](m)
	require.True(t, ok)
	minSoc, maxSoc := limiter.GetSocLimits()
	assert.Equal(t, []float64{20, 100}, []float64{minSoc, maxSoc})

	voltages, ok := api.Cap[api.PhaseVoltages](m)
	require.True(t, ok)
	u1, u2, u3, err := voltages.Voltages()
	require.NoError(t, err)
	assert.Equal(t, []float64{119.6, 120.6, 0}, []float64{u1, u2, u3})

	_, ok = api.Cap[api.PhaseCurrents](m)
	assert.False(t, ok)

	_, ok = api.Cap[api.BatteryController](m)
	assert.False(t, ok, "no control without secret")
}

// basic connections report neither capacity nor backup reserve
func TestPyPowerwallBatteryUnknownValues(t *testing.T) {
	s := newPypowerwallServer(t)
	s.responses["/api/system_status"] = `{}`
	s.responses["/api/operation"] = `{"real_mode": null, "backup_reserve_percent": null}`

	s.responses["/freq"] = `{}`
	m := s.meter(t, "battery", nil)

	_, ok := api.Cap[api.BatteryCapacity](m)
	assert.False(t, ok)
	_, ok = api.Cap[api.PhaseVoltages](m)
	assert.False(t, ok)

	limiter, ok := api.Cap[api.BatterySocLimiter](m)
	require.True(t, ok)
	minSoc, maxSoc := limiter.GetSocLimits()
	assert.Equal(t, []float64{20, 100}, []float64{minSoc, maxSoc}, "configured reserve if unknown")
}

func TestPyPowerwallBatteryControl(t *testing.T) {
	s := newPypowerwallServer(t)
	m := s.meter(t, "battery", map[string]any{"controlSecret": "secret", "minsoc": 15})

	ctrl, ok := api.Cap[api.BatteryController](m)
	require.True(t, ok)
	assert.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHold, api.BatteryCharge}, ctrl.BatteryModes())

	tests := []struct {
		mode api.BatteryMode
		want string
	}{
		{mode: api.BatteryNormal, want: `{"value":15}`},
		{mode: api.BatteryHold, want: `{"value":46}`}, // current soc as integer
		{mode: api.BatteryCharge, want: `{"value":100}`},
	}

	for _, tc := range tests {
		t.Run(tc.mode.String(), func(t *testing.T) {
			require.NoError(t, ctrl.SetBatteryMode(tc.mode))
			assert.Equal(t, "Bearer secret", s.auth)
			assert.JSONEq(t, tc.want, s.body)
		})
	}

	s.status = http.StatusServiceUnavailable
	assert.Error(t, ctrl.SetBatteryMode(api.BatteryHold))
}

func TestPyPowerwallPVMaxACPower(t *testing.T) {
	s := newPypowerwallServer(t)

	_, ok := api.Cap[api.MaxACPowerGetter](s.meter(t, "pv", nil))
	assert.False(t, ok)

	m, ok := api.Cap[api.MaxACPowerGetter](s.meter(t, "pv", map[string]any{"maxacpower": 11520}))
	require.True(t, ok)
	assert.Equal(t, 11520.0, m.MaxACPower())
}

func TestPyPowerwallBatteryLimits(t *testing.T) {
	tests := []struct {
		name              string
		status            string
		config            map[string]any
		ok                bool
		charge, discharge float64
	}{
		{"none", `{"max_charge_power": null, "max_apparent_power": null}`, nil, false, 0, 0},
		{"config incomplete", `{}`, map[string]any{"maxchargepower": 5000}, false, 0, 0},
		{"config", `{}`, map[string]any{"maxchargepower": 5000, "maxdischargepower": 10000}, true, 5000, 10000},
		{"apparent power", `{"max_apparent_power": 10800}`, nil, true, 10800, 10800},
		{"charge discharge", `{"max_charge_power": 9000, "max_discharge_power": 8000, "max_apparent_power": 10800}`, nil, true, 9000, 8000},
		{"config overrides", `{"max_charge_power": 9000, "max_discharge_power": 8000}`, map[string]any{"maxchargepower": 5000}, true, 5000, 8000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newPypowerwallServer(t)
			s.responses["/api/system_status"] = tc.status

			m, ok := api.Cap[api.BatteryPowerLimiter](s.meter(t, "battery", tc.config))
			require.Equal(t, tc.ok, ok)
			if !ok {
				return
			}

			charge, discharge := m.GetPowerLimits()
			assert.Equal(t, tc.charge, charge)
			assert.Equal(t, tc.discharge, discharge)
		})
	}
}

func TestPyPowerwallBatteryChargeMaxSoc(t *testing.T) {
	s := newPypowerwallServer(t)
	m := s.meter(t, "battery", map[string]any{"controlSecret": "secret", "chargemaxsoc": 80})

	limiter, ok := api.Cap[api.BatterySocLimiter](m)
	require.True(t, ok)
	minSoc, maxSoc := limiter.GetSocLimits()
	assert.Equal(t, 20.0, minSoc)
	assert.Equal(t, 100.0, maxSoc, "charge target is not an upper limit")

	ctrl, ok := api.Cap[api.BatteryController](m)
	require.True(t, ok)
	require.NoError(t, ctrl.SetBatteryMode(api.BatteryCharge))
	assert.JSONEq(t, `{"value":80}`, s.body)
}

func TestPyPowerwallTLS(t *testing.T) {
	s := newPypowerwallServer(t)
	srv := httptest.NewTLSServer(s.Config.Handler)
	t.Cleanup(srv.Close)

	_, err := NewPyPowerwallFromConfig(map[string]any{"uri": srv.URL, "usage": "grid"})
	assert.ErrorContains(t, err, "certificate")

	_, err = NewPyPowerwallFromConfig(map[string]any{"uri": srv.URL, "usage": "grid", "insecure": true})
	assert.NoError(t, err)
}

func TestPyPowerwallBatteryReserveReadError(t *testing.T) {
	s := newPypowerwallServer(t)
	m := s.meter(t, "battery", map[string]any{"minsoc": 15, "cache": "0s"})

	limiter, ok := api.Cap[api.BatterySocLimiter](m)
	require.True(t, ok)

	delete(s.responses, "/api/operation")
	minSoc, _ := limiter.GetSocLimits()
	assert.Equal(t, 15.0, minSoc, "configured reserve before first read")

	s.responses["/api/operation"] = `{"backup_reserve_percent": 30.0}`
	minSoc, _ = limiter.GetSocLimits()
	assert.Equal(t, 30.0, minSoc)

	delete(s.responses, "/api/operation")
	minSoc, _ = limiter.GetSocLimits()
	assert.Equal(t, 30.0, minSoc, "last known reserve")

	s.responses["/api/operation"] = `{"backup_reserve_percent": null}`
	minSoc, _ = limiter.GetSocLimits()
	assert.Equal(t, 30.0, minSoc, "last known reserve if null")
}

func TestPyPowerwallBatteryReserveRefresh(t *testing.T) {
	s := newPypowerwallServer(t)
	m := s.meter(t, "battery", map[string]any{"controlSecret": "secret", "cache": "1h"})

	limiter, ok := api.Cap[api.BatterySocLimiter](m)
	require.True(t, ok)
	minSoc, _ := limiter.GetSocLimits()
	assert.Equal(t, 20.0, minSoc)

	ctrl, ok := api.Cap[api.BatteryController](m)
	require.True(t, ok)

	s.responses["/api/operation"] = `{"backup_reserve_percent": 100.0}`
	require.NoError(t, ctrl.SetBatteryMode(api.BatteryCharge))

	minSoc, _ = limiter.GetSocLimits()
	assert.Equal(t, 100.0, minSoc, "reserve refreshed after write")
}

func TestPyPowerwallBatteryReserveRange(t *testing.T) {
	s := newPypowerwallServer(t)

	for _, config := range []map[string]any{
		{"minsoc": 101},
		{"minsoc": -1},
		{"chargemaxsoc": 110},
	} {
		config["uri"], config["usage"] = s.URL, "battery"
		_, err := NewPyPowerwallFromConfig(config)
		assert.Error(t, err, config)
	}

	s.responses["/api/system_status/soe"] = `{"percentage": 85.5}`
	ctrl, ok := api.Cap[api.BatteryController](s.meter(t, "battery", map[string]any{"controlSecret": "secret"}))
	require.True(t, ok)

	require.NoError(t, ctrl.SetBatteryMode(api.BatteryHold))
	assert.JSONEq(t, `{"value":86}`, s.body, "local connections accept 81-99")

	s.meter(t, "battery", map[string]any{"controlSecret": "secret", "minsoc": 90, "chargemaxsoc": 95})
}
