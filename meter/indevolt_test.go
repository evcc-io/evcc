package meter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

type indevoltWrite struct {
	Function int   `json:"f"`
	Point    int   `json:"t"`
	Values   []int `json:"v"`
}

func indevoltMeter(t *testing.T, values map[string]any, endpoint string) api.Meter {
	t.Helper()
	name := values["template"].(string)
	tmpl, err := templates.ByName(templates.Meter, name)
	require.NoError(t, err)
	values["host"] = "localhost"
	b, _, err := tmpl.RenderResult(templates.Meter, templates.RenderModeInstance, values)
	require.NoError(t, err)
	b = []byte(strings.ReplaceAll(string(b), "http://localhost:8080", endpoint))
	var instance templates.Instance
	require.NoError(t, yaml.Unmarshal(b, &instance))
	m, err := NewFromConfig(t.Context(), instance.Type, instance.Other)
	require.NoError(t, err)
	return m
}

func TestIndevoltModes(t *testing.T) {
	for _, template := range []string{"indevolt-battery", "indevolt-hybrid"} {
		t.Run(template, func(t *testing.T) {
			var mu sync.Mutex
			var writes []indevoltWrite
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/rpc/Indevolt.SetData" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				var write indevoltWrite
				if err := json.Unmarshal([]byte(r.URL.Query().Get("config")), &write); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				mu.Lock()
				writes = append(writes, write)
				mu.Unlock()
				_, _ = w.Write([]byte(`{"result":true}`))
			}))
			defer srv.Close()
			values := map[string]any{"template": template, "usage": "battery", "minsoc": 5, "maxsoc": 80, "maxchargepower": 1000, "user": "testuser", "password": "testpassword"}
			if template == "indevolt-hybrid" {
				values["controllable"] = true
			}
			m := indevoltMeter(t, values, srv.URL)
			ctrl, ok := api.Cap[api.BatteryController](m)
			require.True(t, ok)
			require.Equal(t, []api.BatteryMode{api.BatteryNormal, api.BatteryHoldDischarge, api.BatteryCharge}, ctrl.BatteryModes())
			for _, mode := range ctrl.BatteryModes() {
				require.NoError(t, ctrl.SetBatteryMode(mode))
			}
			mu.Lock()
			defer mu.Unlock()
			require.Equal(t, []indevoltWrite{
				{16, 1143, []int{0}}, {16, 47005, []int{1}},
				{16, 47005, []int{4}}, {16, 47015, []int{0, 50, 5}},
				{16, 1143, []int{1}}, {16, 47005, []int{4}}, {16, 47015, []int{1, 1000, 80}},
			}, writes)
		})
	}
}

func TestIndevoltHybridMeasurements(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rpc/Indevolt.GetData" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"2101":500,"2108":200,"2107":10,"1501":700,"1505":2500,"6000":123,"6002":50,"6007":20}`))
	}))
	defer srv.Close()
	for _, tc := range []struct {
		usage         string
		controllable  bool
		power, energy float64
	}{
		{"battery", false, 123, 20}, {"grid", true, 300, 10}, {"pv", true, 700, 2.5},
	} {
		t.Run(tc.usage, func(t *testing.T) {
			m := indevoltMeter(t, map[string]any{"template": "indevolt-hybrid", "usage": tc.usage, "controllable": tc.controllable, "maxchargepower": 1000}, srv.URL)
			require.False(t, api.HasCap[api.BatteryController](m))
			power, err := m.CurrentPower()
			require.NoError(t, err)
			require.Equal(t, tc.power, power)
			energy, ok := api.Cap[api.MeterEnergy](m)
			require.True(t, ok)
			value, err := energy.TotalEnergy()
			require.NoError(t, err)
			require.Equal(t, tc.energy, value)
			if tc.usage == "battery" {
				battery, ok := api.Cap[api.Battery](m)
				require.True(t, ok)
				soc, err := battery.Soc()
				require.NoError(t, err)
				require.Equal(t, 50.0, soc)
			}
		})
	}
}

func TestIndevoltHybridControlPower(t *testing.T) {
	tmpl, err := templates.ByName(templates.Meter, "indevolt-hybrid")
	require.NoError(t, err)
	for _, power := range []int{0, 49, 50, 1000} {
		_, _, err := tmpl.RenderResult(templates.Meter, templates.RenderModeInstance, map[string]any{"usage": "battery", "host": "localhost", "controllable": true, "maxchargepower": power})
		if power < 50 {
			require.ErrorContains(t, err, "maxchargepower must be at least 50 W")
		} else {
			require.NoError(t, err)
		}
	}
}
