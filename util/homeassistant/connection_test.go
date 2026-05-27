package homeassistant

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				body, _ := io.ReadAll(r.Body)
				gotBody = string(body)
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

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
