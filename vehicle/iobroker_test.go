package vehicle

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIobrokerTemplate verifies soc reading and the statusA/B/C value mapping
func TestIobrokerTemplate(t *testing.T) {
	// datapoint id is the value it reports
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := strings.CutSuffix(strings.TrimPrefix(r.URL.Path, "/rest-api/v1/state/"), "/plain")
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(id))
	}))
	defer srv.Close()

	for _, tc := range []struct {
		status string
		res    api.ChargeStatus
		err    bool
	}{
		{"CHARGINGACTIVE", api.StatusC, false},
		{"plugged", api.StatusB, false},
		{"A", api.StatusA, false},
		{"unknown", api.StatusNone, true},
	} {
		t.Run(tc.status, func(t *testing.T) {
			v, err := NewFromConfig(t.Context(), "template", map[string]any{
				"template": "iobroker",
				"uri":      srv.URL,
				"soc":      "42",
				"status":   tc.status,
				"statusB":  "CONNECTED, plugged",
				"statusC":  "CHARGINGACTIVE",
			})
			require.NoError(t, err)

			soc, err := v.Soc()
			require.NoError(t, err)
			assert.Equal(t, 42.0, soc)

			cs, ok := api.Cap[api.ChargeState](v)
			require.True(t, ok)

			s, err := cs.Status()
			if tc.err {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.res, s)
		})
	}
}
