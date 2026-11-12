package meter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
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
			m := indevoltMeter(t, values, srv.URL)
			mu.Lock()
			require.Empty(t, writes)
			mu.Unlock()
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
	registers := map[int]float64{2101: 500, 2108: 200, 2107: 10, 1501: 700, 1505: 2500, 6000: 123, 6002: 50, 6007: 20}
	for _, tc := range []struct {
		usage         string
		power, energy float64
		queries       [][]int
	}{
		{"battery", 123, 20, [][]int{{6000}, {6007}, {6002}}},
		{"grid", 300, 10, [][]int{{2101, 2108}, {2107}}},
		{"pv", 700, 2.5, [][]int{{1501}, {1505}}},
	} {
		t.Run(tc.usage, func(t *testing.T) {
			var mu sync.Mutex
			var queries [][]int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/rpc/Indevolt.GetData" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				var config struct {
					Points []int `json:"t"`
				}
				if err := json.Unmarshal([]byte(r.URL.Query().Get("config")), &config); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				mu.Lock()
				queries = append(queries, config.Points)
				mu.Unlock()
				response := make(map[int]float64, len(config.Points))
				for _, point := range config.Points {
					if value, ok := registers[point]; ok {
						response[point] = value
					} else {
						t.Errorf("unexpected register: %d", point)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Error(err)
				}
			}))
			defer srv.Close()
			m := indevoltMeter(t, map[string]any{"template": "indevolt-hybrid", "usage": tc.usage, "maxchargepower": 1000}, srv.URL)
			require.Equal(t, tc.usage == "battery", api.HasCap[api.BatteryController](m))
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
			mu.Lock()
			defer mu.Unlock()
			require.Equal(t, tc.queries, queries)
		})
	}
}

func TestIndevoltControlPower(t *testing.T) {
	for _, template := range []struct{ name, defaultPower string }{
		{"indevolt-battery", "2400"},
		{"indevolt-hybrid", "1200"},
	} {
		t.Run(template.name, func(t *testing.T) {
			tmpl, err := templates.ByName(templates.Meter, template.name)
			require.NoError(t, err)
			for _, tc := range []struct {
				name  string
				power any
				want  string
			}{
				{"default", nil, template.defaultPower},
				{"override", 1000, "1000"},
				{"higher override", 2400, "2400"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					values := map[string]any{"usage": "battery", "host": "localhost"}
					if tc.power != nil {
						values["maxchargepower"] = tc.power
					}
					b, _, err := tmpl.RenderResult(templates.Meter, templates.RenderModeInstance, values)
					require.NoError(t, err)
					require.Contains(t, string(b), "maxchargepower: "+tc.want)
					var config struct {
						BatteryMode struct {
							Switch []struct {
								Case int
								Set  struct {
									Set []struct{ URI string }
								}
							}
						}
					}
					require.NoError(t, yaml.Unmarshal(b, &config))
					require.Len(t, config.BatteryMode.Switch, 3)
					charge := config.BatteryMode.Switch[2]
					require.Equal(t, 3, charge.Case)
					require.Len(t, charge.Set.Set, 3)
					require.Contains(t, charge.Set.Set[2].URI, `"v":[1,`+tc.want+`,100]`)
				})
			}
		})
	}
}

func TestIndevoltRejectedWrite(t *testing.T) {
	for _, template := range []string{"indevolt-battery", "indevolt-hybrid"} {
		t.Run(template, func(t *testing.T) {
			for _, body := range []string{`{"result":false}`, `{}`, `not json`} {
				t.Run(body, func(t *testing.T) {
					for failedStep := 1; failedStep <= 3; failedStep++ {
						t.Run(fmt.Sprint(failedStep), func(t *testing.T) {
							var mu sync.Mutex
							var points []int
							reject := true
							srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
								var write indevoltWrite
								if err := json.Unmarshal([]byte(r.URL.Query().Get("config")), &write); err != nil {
									t.Error(err)
									w.WriteHeader(http.StatusBadRequest)
									return
								}
								mu.Lock()
								points = append(points, write.Point)
								fail := reject && len(points) == failedStep
								mu.Unlock()
								response := `{"result":true}`
								if fail {
									response = body
								}
								_, _ = w.Write([]byte(response))
							}))
							defer srv.Close()
							values := map[string]any{"template": template, "usage": "battery", "maxchargepower": 1000, "minsoc": 5, "maxsoc": 80}
							m := indevoltMeter(t, values, srv.URL)
							ctrl, ok := api.Cap[api.BatteryController](m)
							require.True(t, ok)
							require.Error(t, ctrl.SetBatteryMode(api.BatteryCharge))
							mu.Lock()
							got := slices.Clone(points)
							reject, points = false, nil
							mu.Unlock()
							require.Equal(t, []int{1143, 47005, 47015}[:failedStep], got)
							require.NoError(t, ctrl.SetBatteryMode(api.BatteryCharge))
							mu.Lock()
							defer mu.Unlock()
							require.Equal(t, []int{1143, 47005, 47015}, points)
						})
					}
				})
			}
		})
	}
}
