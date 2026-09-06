package charger

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type thermostatTestHA struct {
	mu           sync.Mutex
	server       *httptest.Server
	unit         string
	state        string
	attributes   map[string]any
	power        string
	powerUnit    string
	apply        bool
	dropResponse bool
	writes       []float64
}

func newThermostatTestHA(t *testing.T) *thermostatTestHA {
	t.Helper()
	h := &thermostatTestHA{
		unit: "°C", state: "heat", apply: true, power: "700", powerUnit: "W",
		attributes: map[string]any{
			"temperature": 20.0, "current_temperature": 19.8,
			"min_temp": 5.0, "max_temp": 30.0,
			"supported_features": 1, "hvac_modes": []string{"heat"}, "hvac_action": "heating",
		},
	}
	h.server = httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		defer h.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		var response any
		switch r.Method + " " + r.URL.Path {
		case "GET /api/config":
			response = map[string]any{"unit_system": map[string]string{"temperature": h.unit}}
		case "GET /api/states/climate.room":
			response = map[string]any{"entity_id": "climate.room", "state": h.state, "attributes": h.attributes}
		case "GET /api/states/sensor.power":
			response = map[string]any{"entity_id": "sensor.power", "state": h.power,
				"attributes": map[string]string{"unit_of_measurement": h.powerUnit}}
		case "POST /api/services/climate/set_temperature":
			var body map[string]any
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			assert.Len(t, body, 2)
			assert.Equal(t, "climate.room", body["entity_id"])
			target, ok := body["temperature"].(float64)
			if !assert.True(t, ok) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			h.writes = append(h.writes, target)
			if h.apply {
				h.attributes["temperature"] = target
			}
			if h.dropResponse {
				conn, _, err := w.(http.Hijacker).Hijack()
				if assert.NoError(t, err) {
					assert.NoError(t, conn.Close())
				}
				return
			}
			response = []any{}
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	h.server.Start()
	return h
}

func (h *thermostatTestHA) change(f func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	f()
}

func (h *thermostatTestHA) commands() []float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.writes)
}

func (h *thermostatTestHA) charger(t *testing.T, boost float64) *HomeAssistantThermostat {
	t.Helper()
	c, err := NewHomeAssistantThermostatFromConfig(map[string]any{
		"uri": h.server.URL, "entity": "climate.room", "boost": boost, "power": "sensor.power",
	})
	require.NoError(t, err)
	thermostat := c.(*HomeAssistantThermostat)
	thermostat.conn.Client.Transport = http.DefaultTransport
	return thermostat
}

func thermostatTestDatabase(t *testing.T) func() {
	t.Helper()
	previous := db.Instance
	path := filepath.Join(t.TempDir(), "evcc.db")
	require.NoError(t, db.NewInstance("sqlite", path))
	t.Cleanup(func() {
		require.NoError(t, db.Close())
		db.Instance = previous
	})
	return func() {
		t.Helper()
		require.NoError(t, db.Close())
		require.NoError(t, db.NewInstance("sqlite", path))
	}
}

func TestHomeAssistantThermostatBoostRestore(t *testing.T) {
	for _, tc := range []struct {
		name     string
		baseline float64
		boost    float64
	}{
		{"half degree", 20, 0.5},
		{"fractional baseline", 20.1, 0.2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			thermostatTestDatabase(t)
			h := newThermostatTestHA(t)
			h.change(func() { h.attributes["temperature"] = tc.baseline })
			c := h.charger(t, tc.boost)
			status, err := c.Status()
			require.NoError(t, err)
			assert.Equal(t, api.StatusB, status)
			enabled, err := c.Enabled()
			require.NoError(t, err)
			assert.False(t, enabled)
			assert.Empty(t, h.commands(), "construction and getters must not actuate")
			require.NoError(t, c.Enable(true))
			require.NoError(t, c.Enable(true))
			status, err = c.Status()
			require.NoError(t, err)
			assert.Equal(t, api.StatusC, status)
			require.NoError(t, c.Enable(false))
			require.NoError(t, c.Enable(false))
			assert.Equal(t, []float64{tc.baseline + tc.boost, tc.baseline}, h.commands())
			enabled, err = c.Enabled()
			require.NoError(t, err)
			assert.False(t, enabled)
			assert.False(t, settings.Exists(c.key()))
		})
	}
}

func TestHomeAssistantThermostatSharedTarget(t *testing.T) {
	thermostatTestDatabase(t)
	h := newThermostatTestHA(t)
	first, second := h.charger(t, 0.5), h.charger(t, 1)
	require.NoError(t, first.Enable(true))
	require.NoError(t, second.Enable(true))
	assert.Equal(t, []float64{20.5}, h.commands())
	require.NoError(t, second.Enable(false))
	assert.Equal(t, []float64{20.5, 20}, h.commands())
}

func TestHomeAssistantThermostatConcurrentBoost(t *testing.T) {
	thermostatTestDatabase(t)
	h := newThermostatTestHA(t)
	first, second := h.charger(t, 0.5), h.charger(t, 1)
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			c := first
			if i%2 != 0 {
				c = second
			}
			assert.NoError(t, c.Enable(true))
		})
	}
	wg.Wait()
	require.Len(t, h.commands(), 1)
	require.NoError(t, first.Enable(false))
	assert.Equal(t, 20.0, h.commands()[1])
}

func TestHomeAssistantThermostatTinyBoost(t *testing.T) {
	thermostatTestDatabase(t)
	h := newThermostatTestHA(t)
	for _, boost := range []float64{1e-20, 1e-8} {
		c := h.charger(t, boost)
		assert.Error(t, c.Enable(true))
		assert.False(t, settings.Exists(c.key()))
	}
	assert.Empty(t, h.commands())
}

func TestHomeAssistantThermostatRestart(t *testing.T) {
	restart := thermostatTestDatabase(t)
	h := newThermostatTestHA(t)
	c := h.charger(t, 0.5)
	require.NoError(t, c.Enable(true))
	restart()
	c = h.charger(t, 0.5)
	enabled, err := c.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, []float64{20.5}, h.commands())
	require.NoError(t, c.Enable(false))
	assert.Equal(t, []float64{20.5, 20}, h.commands())
}

func TestHomeAssistantThermostatManualHold(t *testing.T) {
	restart := thermostatTestDatabase(t)
	h := newThermostatTestHA(t)
	c := h.charger(t, 0.5)
	require.NoError(t, c.Enable(true))
	_, err := c.Status()
	require.NoError(t, err)
	h.change(func() { h.attributes["temperature"] = 21.0 })
	require.NoError(t, c.Enable(true))
	restart()
	c = h.charger(t, 0.5)
	require.NoError(t, c.Enable(true))
	status, err := c.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusB, status)
	enabled, err := c.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled, "hold suppresses immediate re-acquisition by core")
	require.NoError(t, c.Enable(false))
	assert.Equal(t, []float64{20.5}, h.commands(), "Normal must respect the manual target")
	require.NoError(t, c.Enable(true))
	assert.Equal(t, []float64{20.5, 21.5}, h.commands())
}

func TestHomeAssistantThermostatDelayedBoost(t *testing.T) {
	thermostatTestDatabase(t)
	h := newThermostatTestHA(t)
	h.change(func() { h.apply = false })
	c := h.charger(t, 0.5)
	require.NoError(t, c.Enable(true))
	require.NoError(t, c.Enable(true))
	assert.Error(t, c.Enable(false))
	assert.True(t, settings.Exists(c.key()))
	assert.Equal(t, []float64{20.5}, h.commands())
	h.change(func() {
		h.attributes["temperature"] = 20.5
		h.apply = true
	})
	require.NoError(t, c.Enable(false))
	assert.Equal(t, []float64{20.5, 20}, h.commands())
}

func TestHomeAssistantThermostatDelayedBoostManualChange(t *testing.T) {
	restart := thermostatTestDatabase(t)
	h := newThermostatTestHA(t)
	h.change(func() { h.apply = false })
	c := h.charger(t, 0.5)
	require.NoError(t, c.Enable(true))
	h.change(func() { h.attributes["temperature"] = 21.0 })
	assert.Error(t, c.Enable(false), "a manual change does not acknowledge the outstanding boost")
	assert.True(t, settings.Exists(c.key()))
	restart()
	c = h.charger(t, 0.5)
	h.change(func() { h.attributes["temperature"] = 20.5 })
	assert.Error(t, c.Enable(false), "the late boost must not overwrite the manual choice again")
	assert.True(t, settings.Exists(c.key()))
	assert.Equal(t, []float64{20.5}, h.commands())
	h.change(func() { h.attributes["temperature"] = 21.0 })
	require.NoError(t, c.Enable(false))
	assert.False(t, settings.Exists(c.key()))
	assert.Equal(t, []float64{20.5}, h.commands())
}

func TestHomeAssistantThermostatManualOff(t *testing.T) {
	thermostatTestDatabase(t)
	h := newThermostatTestHA(t)
	c := h.charger(t, 0.5)
	require.NoError(t, c.Enable(true))
	h.change(func() { h.state = "off" })
	status, err := c.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusB, status)
	enabled, err := c.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Error(t, c.Enable(false))
	assert.True(t, settings.Exists(c.key()))
	assert.Equal(t, []float64{20.5}, h.commands())
	h.change(func() { h.attributes["temperature"] = 20.0 })
	require.NoError(t, c.Enable(false))
	assert.False(t, settings.Exists(c.key()))
	assert.Equal(t, []float64{20.5}, h.commands())
}

func TestHomeAssistantThermostatLostResponse(t *testing.T) {
	for _, duringRestore := range []bool{false, true} {
		name := "boost"
		if duringRestore {
			name = "restore"
		}
		t.Run(name, func(t *testing.T) {
			restart := thermostatTestDatabase(t)
			h := newThermostatTestHA(t)
			c := h.charger(t, 0.5)
			if duringRestore {
				require.NoError(t, c.Enable(true))
			}
			h.change(func() { h.dropResponse = true })
			assert.Error(t, c.Enable(!duringRestore))
			assert.True(t, settings.Exists(c.key()))
			h.change(func() { h.dropResponse = false })
			restart()
			c = h.charger(t, 0.5)
			require.NoError(t, c.Enable(false))
			assert.Equal(t, []float64{20.5, 20}, h.commands(), "do not repeat an already applied command")
			assert.False(t, settings.Exists(c.key()))
		})
	}
}

func TestHomeAssistantThermostatDelayedRestore(t *testing.T) {
	restart := thermostatTestDatabase(t)
	h := newThermostatTestHA(t)
	c := h.charger(t, 0.5)
	require.NoError(t, c.Enable(true))
	h.change(func() { h.apply = false })
	assert.Error(t, c.Enable(false))
	assert.Error(t, c.Enable(true), "a pending restore cannot start a new boost")
	assert.True(t, settings.Exists(c.key()))
	restart()
	c = h.charger(t, 0.5)
	h.change(func() { h.attributes["temperature"] = 20.0 })
	require.NoError(t, c.Enable(false))
	assert.Equal(t, []float64{20.5, 20}, h.commands())
	assert.False(t, settings.Exists(c.key()))
}

func TestHomeAssistantThermostatRestoreManualChange(t *testing.T) {
	thermostatTestDatabase(t)
	h := newThermostatTestHA(t)
	c := h.charger(t, 0.5)
	require.NoError(t, c.Enable(true))
	h.change(func() { h.apply = false })
	assert.Error(t, c.Enable(false))
	h.change(func() { h.attributes["temperature"] = 21.0 })
	assert.Error(t, c.Enable(false), "an unrelated target does not confirm restoration")
	assert.True(t, settings.Exists(c.key()))
	assert.Equal(t, []float64{20.5, 20}, h.commands())
}

func TestHomeAssistantThermostatInvalidState(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*thermostatTestHA)
	}{
		{"fahrenheit", func(h *thermostatTestHA) { h.unit = "°F" }},
		{"fahrenheit attribute", func(h *thermostatTestHA) { h.attributes["unit_of_measurement"] = "°F" }},
		{"unknown", func(h *thermostatTestHA) { h.state = "unknown" }},
		{"unavailable", func(h *thermostatTestHA) { h.state = "unavailable" }},
		{"cooling", func(h *thermostatTestHA) { h.state = "cool" }},
		{"off", func(h *thermostatTestHA) { h.state = "off" }},
		{"readonly", func(h *thermostatTestHA) { h.attributes["supported_features"] = 0 }},
		{"range only", func(h *thermostatTestHA) { h.attributes["supported_features"] = 2 }},
		{"missing target", func(h *thermostatTestHA) { delete(h.attributes, "temperature") }},
		{"missing minimum", func(h *thermostatTestHA) { delete(h.attributes, "min_temp") }},
		{"missing maximum", func(h *thermostatTestHA) { delete(h.attributes, "max_temp") }},
		{"inverted limits", func(h *thermostatTestHA) { h.attributes["min_temp"] = 31.0 }},
		{"target out of range", func(h *thermostatTestHA) { h.attributes["temperature"] = 31.0 }},
		{"boost out of range", func(h *thermostatTestHA) { h.attributes["temperature"] = 30.0 }},
		{"invalid step", func(h *thermostatTestHA) { h.attributes["target_temp_step"] = 0.0 }},
		{"step mismatch", func(h *thermostatTestHA) { h.attributes["target_temp_step"] = 1.0 }},
		{"non numeric target", func(h *thermostatTestHA) { h.attributes["temperature"] = "NaN" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			thermostatTestDatabase(t)
			h := newThermostatTestHA(t)
			h.change(func() { tc.change(h) })
			c := h.charger(t, 0.5)
			assert.Error(t, c.Enable(true))
			assert.Empty(t, h.commands())
			assert.False(t, settings.Exists(c.key()))
		})
	}
}

func TestHomeAssistantThermostatInvalidConfig(t *testing.T) {
	for _, tc := range []struct {
		name  string
		key   string
		value any
	}{
		{"zero boost", "boost", 0.0},
		{"negative boost", "boost", -0.5},
		{"NaN boost", "boost", math.NaN()},
		{"infinite boost", "boost", math.Inf(1)},
		{"wrong entity", "entity", "switch.room"},
		{"empty entity", "entity", "climate."},
		{"entity path", "entity", "climate.room/other"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := map[string]any{"uri": "http://127.0.0.1", "entity": "climate.room", "boost": 0.5}
			config[tc.key] = tc.value
			_, err := NewHomeAssistantThermostatFromConfig(config)
			assert.Error(t, err)
		})
	}
}

func TestHomeAssistantThermostatPersistenceFailure(t *testing.T) {
	for _, missing := range []bool{true, false} {
		name := "write failure"
		if missing {
			name = "no database"
		}
		t.Run(name, func(t *testing.T) {
			thermostatTestDatabase(t)
			h := newThermostatTestHA(t)
			c := h.charger(t, 0.5)
			instance := db.Instance
			if missing {
				db.Instance = nil
				defer func() { db.Instance = instance }()
			} else {
				require.NoError(t, db.Instance.Exec("PRAGMA query_only = ON").Error)
			}
			assert.Error(t, c.Enable(true))
			assert.Empty(t, h.commands(), "persistence must succeed before a command is sent")
			assert.False(t, settings.Exists(c.key()))
			if !missing {
				require.NoError(t, db.Instance.Exec("PRAGMA query_only = OFF").Error)
				require.NoError(t, c.Enable(true))
				assert.Equal(t, []float64{20.5}, h.commands())
			}
		})
	}
}

func TestHomeAssistantThermostatMemoryDatabase(t *testing.T) {
	for _, dsn := range []string{":memory:", "file:thermostat-memory?mode=memory&cache=shared"} {
		t.Run(dsn, func(t *testing.T) {
			thermostatTestDatabase(t)
			require.NoError(t, db.Close())
			require.NoError(t, db.NewInstance("sqlite", dsn))
			h := newThermostatTestHA(t)
			c := h.charger(t, 0.5)
			assert.Error(t, c.Enable(true))
			assert.Empty(t, h.commands())
			assert.False(t, settings.Exists(c.key()))
		})
	}
}

func TestHomeAssistantThermostatMeasurements(t *testing.T) {
	h := newThermostatTestHA(t)
	c := h.charger(t, 0.5)
	temperature, err := c.Soc()
	require.NoError(t, err)
	assert.Equal(t, 19.8, temperature)
	meter, ok := api.Cap[api.Meter](c)
	require.True(t, ok)
	for _, tc := range []struct {
		value string
		unit  string
		want  float64
		err   bool
	}{
		{"700", "W", 700, false},
		{"0.7", "kW", 700, false},
		{"700", "kWh", 0, true},
		{"700", "", 0, true},
		{"unknown", "W", 0, true},
		{"NaN", "W", 0, true},
		{"+Inf", "W", 0, true},
		{"-10", "W", 0, true},
	} {
		t.Run(tc.value+tc.unit, func(t *testing.T) {
			h.change(func() { h.power, h.powerUnit = tc.value, tc.unit })
			power, err := meter.CurrentPower()
			if tc.err {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, power)
			}
		})
	}
	h.change(func() { delete(h.attributes, "current_temperature") })
	_, err = c.Soc()
	assert.Error(t, err)
	assert.Empty(t, h.commands())
}
