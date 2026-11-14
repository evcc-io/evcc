package meter

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/lgpcs"
	"github.com/stretchr/testify/require"
)

func TestLgEssBatteryHold(t *testing.T) {
	var backupSoc string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/v1/user/setting/login"):
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "auth_key": "k"})
		case strings.HasSuffix(r.URL.Path, "/v1/user/essinfo/home"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Statistics": map[string]any{"bat_user_soc": "99.5"},
			})
		case strings.HasSuffix(r.URL.Path, "/v1/user/setting/batt"):
			body, _ := io.ReadAll(r.Body)
			var p map[string]string
			_ = json.Unmarshal(body, &p)
			backupSoc = p["backup_soc"]
			_ = json.NewEncoder(w).Encode(map[string]string{})
		default:
			t.Logf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	m, err := NewLgEss(srv.URL, "battery", "", "pw", time.Millisecond, batteryCapacity{}, batterySocLimits{MinSoc: 20, MaxSoc: 95}, batteryPowerLimits{}, lgpcs.LgEss8)
	require.NoError(t, err)

	lg, ok := m.(*LgEss)
	require.True(t, ok)

	require.NoError(t, lg.batteryMode(batterySocLimits{MinSoc: 20, MaxSoc: 95})(api.BatteryHold))
	require.Equal(t, "100", backupSoc)
}
