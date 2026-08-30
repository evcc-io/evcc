package homeassistant

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConnection(baseURL string) *Connection {
	return &Connection{
		Helper:   request.NewHelper(util.NewLogger("test")),
		instance: &proxyInstance{uri: baseURL},
	}
}

// newStateConnection returns a connection serving state for any entity
func newStateConnection(t *testing.T, state string) *Connection {
	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"entity_id":"sensor.foo","state":%q}`, state)
	}))
	srv.Start()

	return newTestConnection(srv.URL)
}

func TestGetChargeStatus(t *testing.T) {
	states, err := NewStatusMap("not_plugged", "Charging_Stopped, charging_error", "instant_charging")
	require.NoError(t, err)

	tests := []struct {
		state string
		want  api.ChargeStatus
	}{
		{"A", api.StatusA},
		{"connected", api.StatusB},
		{" charging ", api.StatusC},
		{"not_plugged", api.StatusA},
		{"CHARGING_STOPPED", api.StatusB},
		{"charging_error", api.StatusB},
		{"instant_charging", api.StatusC},
		{"paused", api.StatusB},
		{"preparing", api.StatusNone}, // no longer built-in
	}

	for _, tc := range tests {
		t.Run(tc.state, func(t *testing.T) {
			status, err := newStateConnection(t, tc.state).GetChargeStatus("sensor.foo", states)
			assert.Equal(t, tc.want, status)
			if tc.want == api.StatusNone {
				assert.ErrorContains(t, err, "unknown charge status '"+tc.state+"' for entity sensor.foo")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewStatusMapDuplicate(t *testing.T) {
	_, err := NewStatusMap("foo", "foo", "")
	assert.Error(t, err)
}

// TestStatusMapExtendsBuiltin verifies that configured states extend the
// built-in mapping and only override the states they explicitly redefine.
func TestStatusMapExtendsBuiltin(t *testing.T) {
	states, err := NewStatusMap("", "charging", "instant_charging")
	require.NoError(t, err)

	tests := []struct {
		state string
		want  api.ChargeStatus
	}{
		{"charging", api.StatusB},         // redefined
		{"paused", api.StatusB},           // built-in, untouched
		{"c", api.StatusC},                // built-in, untouched
		{"instant_charging", api.StatusC}, // added
	}

	for _, tc := range tests {
		t.Run(tc.state, func(t *testing.T) {
			status, err := newStateConnection(t, tc.state).GetChargeStatus("sensor.foo", states)
			require.NoError(t, err)
			assert.Equal(t, tc.want, status)
		})
	}
}

// TestCallSwitchService_DomainDispatch verifies that CallSwitchService picks
// the correct service per Home Assistant domain — switches use turn_on /
// turn_off, but the stateless button / input_button domains expose only
// `press`. Regression test for evcc-io/evcc#29700.
func TestCallSwitchService_DomainDispatch(t *testing.T) {
	tests := []struct {
		name        string
		entity      string
		turnOn      bool
		wantPath    string
		wantErrText string
	}{
		{"switch turn_on", "switch.foo", true, "/api/services/switch/turn_on", ""},
		{"switch turn_off", "switch.foo", false, "/api/services/switch/turn_off", ""},
		{"button press", "button.tesla_model_x_wake_up", true, "/api/services/button/press", ""},
		{"input_button press", "input_button.bar", true, "/api/services/input_button/press", ""},
		{"button no off", "button.foo", false, "", "entity button.foo has no off action"},
		{"input_button no off", "input_button.bar", false, "", "entity input_button.bar has no off action"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath, gotBody string
			srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				body, _ := io.ReadAll(r.Body)
				gotBody = string(body)
				w.WriteHeader(http.StatusOK)
			}))
			srv.Start()

			err := newTestConnection(srv.URL).CallSwitchService(tc.entity, tc.turnOn)

			if tc.wantErrText != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrText)
				assert.Empty(t, gotPath, "must not call HA when erroring locally")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantPath, gotPath)
			assert.Contains(t, gotBody, `"entity_id":"`+tc.entity+`"`)
		})
	}
}
