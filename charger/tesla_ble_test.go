package charger_test

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/vehicle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTeslaBLEStatusAndDisable(t *testing.T) {
	for _, tc := range []struct {
		state string
		want  api.ChargeStatus
	}{
		{"Charging", api.StatusC},
		{"Starting", api.StatusB},
		{"Stopped", api.StatusB},
		{"NoPower", api.StatusB},
		{"Complete", api.StatusB},
		{"Disconnected", api.StatusA},
		{"Unknown", api.StatusA},
	} {
		t.Run(tc.state, func(t *testing.T) {
			var stops atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/1/vehicles/TESTVIN/vehicle_data":
					assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{
						"response": map[string]any{"response": map[string]any{
							"charge_state": map[string]any{"charging_state": tc.state},
						}},
					}))
				case "/api/1/vehicles/TESTVIN/command/charge_stop":
					assert.Equal(t, http.MethodPost, r.Method)
					stops.Add(1)
					_, err := w.Write([]byte(`{"response":{"result":true}}`))
					assert.NoError(t, err)
				default:
					t.Errorf("unexpected request: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			u, err := url.Parse(server.URL)
			require.NoError(t, err)
			host, port, err := net.SplitHostPort(u.Host)
			require.NoError(t, err)
			v, err := vehicle.NewFromConfig(t.Context(), "template", map[string]any{
				"template": "tesla-ble", "url": u.Scheme + "://" + host, "port": port, "vin": "TESTVIN",
			})
			require.NoError(t, err)
			lp := loadpoint.NewMockAPI(gomock.NewController(t))
			lp.EXPECT().GetVehicle().Return(v).AnyTimes()
			c, err := charger.NewVehicleApiFromConfig(map[string]any{})
			require.NoError(t, err)
			c.(loadpoint.Controller).LoadpointControl(lp)

			status, err := c.Status()
			require.NoError(t, err)
			assert.Equal(t, tc.want, status)
			require.NoError(t, c.Enable(false))
			if tc.want == api.StatusA {
				assert.Zero(t, stops.Load())
			} else {
				assert.EqualValues(t, 1, stops.Load())
			}
		})
	}
}
