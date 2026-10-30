package meter

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePowerwallServer struct {
	mu                sync.Mutex
	secret            string
	reserve           float64
	holds             int
	loginCalls        atomic.Int32
	apparent          float64
	usable            float64
	maxChargePower    float64
	maxDischargePower float64
	reserveNull       bool
	partialFreq       bool
}

func (f *fakePowerwallServer) reserveValue() float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reserve
}

func (f *fakePowerwallServer) holdCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.holds
}

func (f *fakePowerwallServer) loginCount() int {
	return int(f.loginCalls.Load())
}

func (f *fakePowerwallServer) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"din":              "test",
			"start_time":       "2020-01-01 00:00:00 +0000",
			"up_time_seconds":  "1h2m3s",
			"is_new":           false,
			"version":          "1.0",
			"git_hash":         "",
			"commission_count": 0,
			"device_type":      "test",
			"sync_type":        "test",
		})
	})

	mux.HandleFunc("/api/meters/aggregates", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"site": map[string]any{
				"instant_power":   20.0,
				"energy_imported": 3000.0,
				"energy_exported": 4000.0,
				"i_a_current":     1.0,
				"i_b_current":     2.0,
				"i_c_current":     3.0,
			},
			"solar": map[string]any{
				"instant_power":   2243.0,
				"energy_exported": 5000.0,
			},
			"battery": map[string]any{
				"instant_power":   -1200.0,
				"energy_imported": 1000.0,
				"energy_exported": 2000.0,
			},
			"load": map[string]any{
				"instant_power":   2086.75,
				"energy_imported": 6000.0,
			},
		})
	})

	mux.HandleFunc("/api/system_status", func(w http.ResponseWriter, r *http.Request) {
		apparent := f.apparent
		if apparent == 0 {
			apparent = 10000
		}
		usable := f.usable
		if usable == 0 {
			usable = 9200
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"nominal_full_pack_energy":      27000,
			"max_apparent_power":            apparent,
			"inverter_nominal_usable_power": usable,
			"max_charge_power":              f.maxChargePower,
			"max_discharge_power":           f.maxDischargePower,
		})
	})

	mux.HandleFunc("/freq", func(w http.ResponseWriter, r *http.Request) {
		if f.partialFreq {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ISLAND_VL1N_Main": 243.0,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ISLAND_VL1N_Main": 243.0,
			"ISLAND_VL2N_Main": 242.0,
			"ISLAND_VL3N_Main": 241.0,
		})
	})

	mux.HandleFunc("/api/system_status/soe", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"percentage": 50})
	})

	mux.HandleFunc("/api/operation", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.reserveNull {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"real_mode":              "autonomous",
				"backup_reserve_percent": nil,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"real_mode":              "autonomous",
			"backup_reserve_percent": f.reserve,
		})
	})

	mux.HandleFunc("/api/login/Basic", func(w http.ResponseWriter, r *http.Request) {
		f.loginCalls.Add(1)
		http.Error(w, "login not expected", http.StatusInternalServerError)
	})

	mux.HandleFunc("/control/reserve", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+f.secret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var body struct {
			Value int `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Value < 0 || body.Value > 100 {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		f.mu.Lock()
		f.reserve = float64(body.Value)
		f.mu.Unlock()
	})

	mux.HandleFunc("/control/reserve_hold", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+f.secret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		f.mu.Lock()
		f.holds++
		f.reserve = 50
		f.mu.Unlock()
	})

	return mux
}

func newTestPowerWallServer(t *testing.T, fake *fakePowerwallServer, usage string, extra map[string]any) (*PowerWall, *httptest.Server) {
	t.Helper()

	srv := httptest.NewServer(fake.handler())
	t.Cleanup(srv.Close)

	cfg := map[string]any{
		"uri":           srv.Listener.Addr().String(),
		"usage":         usage,
		"scheme":        "http",
		"skipLogin":     true,
		"controlSecret": "secret",
		"minsoc":        0.0,
		"cache":         time.Millisecond,
	}
	maps.Copy(cfg, extra)

	m, err := NewPowerWallFromConfig(cfg)
	require.NoError(t, err)

	pw, ok := m.(*PowerWall)
	require.True(t, ok)

	return pw, srv
}

func TestPowerWallServerBatteryReads(t *testing.T) {
	fake := &fakePowerwallServer{secret: "secret", reserve: 20}
	pw, _ := newTestPowerWallServer(t, fake, "battery", nil)

	power, err := pw.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, -1200.0, power)

	b, ok := api.Cap[api.Battery](pw)
	require.True(t, ok)
	soc, err := b.Soc()
	require.NoError(t, err)
	assert.Equal(t, 50.0, soc)

	c, ok := api.Cap[api.BatteryCapacity](pw)
	require.True(t, ok)
	capacity, err := c.Capacity()
	require.NoError(t, err)
	assert.Equal(t, 27.0, capacity)

	sl, ok := api.Cap[api.BatterySocLimiter](pw)
	require.True(t, ok)
	lower, upper, err := sl.GetSocLimits()
	require.NoError(t, err)
	assert.Equal(t, 20.0, lower)
	assert.Equal(t, 100.0, upper)

	pl, ok := api.Cap[api.BatteryPowerLimiter](pw)
	require.True(t, ok)
	charge, discharge, err := pl.GetPowerLimits()
	require.NoError(t, err)
	assert.Equal(t, 9200.0, charge)
	assert.Equal(t, 9200.0, discharge)

	en, ok := api.Cap[api.MeterEnergy](pw)
	require.True(t, ok)
	charged, err := en.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 1.0, charged)

	re, ok := api.Cap[api.MeterReturnEnergy](pw)
	require.True(t, ok)
	discharged, err := re.ReturnEnergy()
	require.NoError(t, err)
	assert.Equal(t, 2.0, discharged)

	assert.Equal(t, 0, fake.loginCount())
}

func TestPowerWallServerReadOnly(t *testing.T) {
	fake := &fakePowerwallServer{secret: "secret", reserve: 20}
	pw, _ := newTestPowerWallServer(t, fake, "battery", map[string]any{"controlSecret": ""})

	_, ok := api.Cap[api.BatteryController](pw)
	assert.False(t, ok)

	_, ok = api.Cap[api.BatterySocLimiter](pw)
	assert.True(t, ok)
}

func TestPowerWallServerGridReads(t *testing.T) {
	fake := &fakePowerwallServer{secret: "secret", reserve: 20}
	pw, _ := newTestPowerWallServer(t, fake, "grid", nil)

	power, err := pw.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 20.0, power)

	en, ok := api.Cap[api.MeterEnergy](pw)
	require.True(t, ok)
	imported, err := en.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 3.0, imported)

	re, ok := api.Cap[api.MeterReturnEnergy](pw)
	require.True(t, ok)
	exported, err := re.ReturnEnergy()
	require.NoError(t, err)
	assert.Equal(t, 4.0, exported)

	currents, ok := api.Cap[api.PhaseCurrents](pw)
	require.True(t, ok)
	a, b, c, err := currents.Currents()
	require.NoError(t, err)
	assert.Equal(t, 1.0, a)
	assert.Equal(t, 2.0, b)
	assert.Equal(t, 3.0, c)

	voltages, ok := api.Cap[api.PhaseVoltages](pw)
	require.True(t, ok)
	l1, l2, l3, err := voltages.Voltages()
	require.NoError(t, err)
	assert.Equal(t, 243.0, l1)
	assert.Equal(t, 242.0, l2)
	assert.Equal(t, 241.0, l3)
}

func TestPowerWallServerPVReads(t *testing.T) {
	fake := &fakePowerwallServer{secret: "secret", reserve: 20}
	pw, _ := newTestPowerWallServer(t, fake, "pv", nil)

	power, err := pw.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 2243.0, power)

	en, ok := api.Cap[api.MeterEnergy](pw)
	require.True(t, ok)
	produced, err := en.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 5.0, produced)
}

func TestPowerWallServerChargeMaxSoc(t *testing.T) {
	fake := &fakePowerwallServer{secret: "secret", reserve: 20}
	pw, _ := newTestPowerWallServer(t, fake, "battery", map[string]any{"chargemaxsoc": 90.0})

	ctrl, ok := api.Cap[api.BatteryController](pw)
	require.True(t, ok)

	require.NoError(t, ctrl.SetBatteryMode(api.BatteryCharge))
	assert.Equal(t, 90.0, fake.reserveValue())
}

func TestPowerWallServerPowerLimitDerating(t *testing.T) {
	fake := &fakePowerwallServer{
		secret:            "secret",
		reserve:           20,
		maxChargePower:    10000,
		maxDischargePower: 10000,
	}
	pw, _ := newTestPowerWallServer(t, fake, "battery", nil)

	lim, ok := api.Cap[api.BatteryPowerLimiter](pw)
	require.True(t, ok)

	charge, discharge, err := lim.GetPowerLimits()
	require.NoError(t, err)
	assert.Equal(t, 9200.0, charge)
	assert.Equal(t, 9200.0, discharge)
}

func TestPowerWallServerPowerLimitBelowUsable(t *testing.T) {
	fake := &fakePowerwallServer{
		secret:            "secret",
		reserve:           20,
		maxChargePower:    9000,
		maxDischargePower: 9100,
	}
	pw, _ := newTestPowerWallServer(t, fake, "battery", nil)

	lim, ok := api.Cap[api.BatteryPowerLimiter](pw)
	require.True(t, ok)

	charge, discharge, err := lim.GetPowerLimits()
	require.NoError(t, err)
	assert.Equal(t, 9000.0, charge)
	assert.Equal(t, 9100.0, discharge)
}

func TestPowerWallServerPowerLimitOverrides(t *testing.T) {
	fake := &fakePowerwallServer{secret: "secret", reserve: 20}
	pw, _ := newTestPowerWallServer(t, fake, "battery", map[string]any{
		"maxchargepower":    8000.0,
		"maxdischargepower": 8500.0,
	})

	lim, ok := api.Cap[api.BatteryPowerLimiter](pw)
	require.True(t, ok)

	charge, discharge, err := lim.GetPowerLimits()
	require.NoError(t, err)
	assert.Equal(t, 8000.0, charge)
	assert.Equal(t, 8500.0, discharge)
}

func TestPowerWallServerPartialPhaseReadings(t *testing.T) {
	fake := &fakePowerwallServer{secret: "secret", reserve: 20, partialFreq: true}
	pw, _ := newTestPowerWallServer(t, fake, "grid", nil)

	_, ok := api.Cap[api.PhaseCurrents](pw)
	assert.False(t, ok)

	_, ok = api.Cap[api.PhaseVoltages](pw)
	assert.False(t, ok)
}

func TestPowerWallServerTLSInsecure(t *testing.T) {
	fake := &fakePowerwallServer{secret: "secret", reserve: 20}
	srv := httptest.NewTLSServer(fake.handler())
	t.Cleanup(srv.Close)

	cfg := map[string]any{
		"uri":           srv.Listener.Addr().String(),
		"usage":         "battery",
		"scheme":        "https",
		"skipLogin":     true,
		"controlSecret": "secret",
		"minsoc":        0.0,
		"cache":         time.Millisecond,
	}

	_, err := NewPowerWallFromConfig(cfg)
	require.Error(t, err)

	cfg["insecure"] = true
	m, err := NewPowerWallFromConfig(cfg)
	require.NoError(t, err)
	pw, ok := m.(*PowerWall)
	require.True(t, ok)

	ctrl, ok := api.Cap[api.BatteryController](pw)
	require.True(t, ok)
	require.NoError(t, ctrl.SetBatteryMode(api.BatteryNormal))
	assert.Equal(t, 0.0, fake.reserveValue())
}

func TestPowerWallServerNullReserve(t *testing.T) {
	fake := &fakePowerwallServer{secret: "secret", reserve: 20, reserveNull: true}
	pw, _ := newTestPowerWallServer(t, fake, "battery", nil)

	lim, ok := api.Cap[api.BatterySocLimiter](pw)
	require.True(t, ok)

	_, _, err := lim.GetSocLimits()
	assert.ErrorContains(t, err, "no backup reserve")
}

func TestPowerWallServerReserveCacheInvalidation(t *testing.T) {
	fake := &fakePowerwallServer{secret: "secret", reserve: 20}
	pw, _ := newTestPowerWallServer(t, fake, "battery", map[string]any{"cache": time.Hour})

	lim, ok := api.Cap[api.BatterySocLimiter](pw)
	require.True(t, ok)

	lower, _, err := lim.GetSocLimits()
	require.NoError(t, err)
	assert.Equal(t, 20.0, lower)

	ctrl, ok := api.Cap[api.BatteryController](pw)
	require.True(t, ok)
	require.NoError(t, ctrl.SetBatteryMode(api.BatteryNormal))

	lower, _, err = lim.GetSocLimits()
	require.NoError(t, err)
	assert.Equal(t, 0.0, lower)
}

func TestPowerWallServerControl(t *testing.T) {
	fake := &fakePowerwallServer{secret: "secret", reserve: 20}
	pw, _ := newTestPowerWallServer(t, fake, "battery", nil)

	ctrl, ok := api.Cap[api.BatteryController](pw)
	require.True(t, ok)

	require.NoError(t, ctrl.SetBatteryMode(api.BatteryNormal))
	assert.Equal(t, 0.0, fake.reserveValue())

	require.NoError(t, ctrl.SetBatteryMode(api.BatteryHold))
	assert.Equal(t, 1, fake.holdCount())
	assert.Equal(t, 50.0, fake.reserveValue())

	require.NoError(t, ctrl.SetBatteryMode(api.BatteryCharge))
	assert.Equal(t, 100.0, fake.reserveValue())

	assert.Error(t, ctrl.SetBatteryMode(api.BatteryMode(42)))
}

func TestPowerWallServerControlErrors(t *testing.T) {
	fake := &fakePowerwallServer{secret: "secret", reserve: 20}
	pw, _ := newTestPowerWallServer(t, fake, "battery", nil)

	assert.ErrorContains(t, pw.setReserve(150), "0-100")
	assert.ErrorContains(t, pw.setReserve(-1), "0-100")

	pw.controlSecret = "wrong"
	assert.ErrorContains(t, pw.setReserve(20), "401")
}

func TestPowerWallAnyFloat(t *testing.T) {
	tests := []struct {
		in   any
		want float64
	}{
		{in: 1.5, want: 1.5},
		{in: float32(2), want: 2},
		{in: 3, want: 3},
		{in: int64(4), want: 4},
		{in: "5.5", want: 5.5},
		{in: nil, want: 0},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.want, anyFloat(tc.in))
	}
}
