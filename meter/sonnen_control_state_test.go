package meter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/require"
)

func TestSonnenChargePowerUpdateOnlyWritesSetpoint(t *testing.T) {
	var requests []string
	fail := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "test-only", r.Header.Get("Auth-Token"))
		requests = append(requests, r.Method+" "+r.URL.Path)
		if fail {
			http.Error(w, "rejected", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()
	m, err := NewFromConfig(t.Context(), "template", map[string]any{
		"template": "sonnenbatterie", "usage": "battery", "host": strings.TrimPrefix(server.URL, "http://"), "token": "test-only",
	})
	require.NoError(t, err)
	setter, ok := api.Cap[api.BatteryChargePowerController](m)
	require.True(t, ok)
	require.NoError(t, setter.SetBatteryChargePower(410))
	require.Equal(t, []string{
		"PUT /api/v2/configurations",
		"POST /api/v2/setpoint/discharge/0",
		"POST /api/v2/setpoint/charge/410",
	}, requests)
	requests = nil
	update, ok := setter.(api.BatteryChargePowerUpdater)
	require.True(t, ok)
	require.NoError(t, update.UpdateBatteryChargePower(432))
	require.NoError(t, update.UpdateBatteryChargePower(400))
	require.Equal(t, []string{
		"POST /api/v2/setpoint/charge/432",
		"POST /api/v2/setpoint/charge/400",
	}, requests)
	requests = nil
	fail = true
	require.Error(t, update.UpdateBatteryChargePower(450))
	require.Equal(t, []string{"POST /api/v2/setpoint/charge/450"}, requests)
}

func TestSonnenActualControlState(t *testing.T) {
	for _, tc := range []struct {
		mode, system string
		want         api.BatteryOperatingMode
	}{
		{`"1"`, "OnGrid", api.BatteryOperatingManual},
		{`2`, "OnGrid", api.BatteryOperatingAuto},
		{`"10"`, "OnGrid", api.BatteryOperatingAuto},
		{`null`, "OnGrid", api.BatteryOperatingUnknown},
		{`"2"`, "OffGrid", api.BatteryOperatingUnknown},
	} {
		t.Run(tc.mode+tc.system, func(t *testing.T) {
			reads := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, "/api/v2/status", r.URL.Path)
				reads++
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"OperatingMode":%s,"SystemStatus":%q,"Pac_total_W":-123,"USOC":51,"BatteryCharging":true,"Timestamp":"2026-10-09 12:00:00"}`, tc.mode, tc.system)
			}))
			defer server.Close()
			m, err := NewFromConfig(t.Context(), "template", map[string]any{
				"template": "sonnenbatterie", "usage": "battery", "host": strings.TrimPrefix(server.URL, "http://"), "token": "test-only",
			})
			require.NoError(t, err)
			reader, ok := api.Cap[api.BatteryControlStateReader](m)
			require.True(t, ok)
			state, err := reader.BatteryControlState()
			require.NoError(t, err)
			require.Equal(t, tc.want, state.Mode)
			require.Equal(t, -123.0, state.Power)
			require.True(t, state.NativeCharging)
			require.Empty(t, state.LeaseID)
			require.False(t, api.HasCap[api.BatteryPVLeaseController](m))
			require.Equal(t, 1, reads)
		})
	}
}

func TestSonnenRestorePreviousAutomaticMode(t *testing.T) {
	for _, fallback := range []string{"self-consumption", "time-of-use"} {
		t.Run(fallback, func(t *testing.T) {
			var modes []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPut, r.Method)
				require.Equal(t, "/api/v2/configurations", r.URL.Path)
				require.Equal(t, "test-only", r.Header.Get("Auth-Token"))
				var body map[string]string
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				modes = append(modes, body["EM_OperatingMode"])
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{}`)
			}))
			defer server.Close()
			m, err := NewFromConfig(t.Context(), "template", map[string]any{
				"template": "sonnenbatterie", "usage": "battery", "host": strings.TrimPrefix(server.URL, "http://"), "token": "test-only", "defaultmode": fallback,
			})
			require.NoError(t, err)
			restore, ok := api.Cap[api.BatteryControlRestorer](m)
			require.True(t, ok)
			require.NoError(t, restore.RestoreBatteryMode("10"))
			require.NoError(t, restore.RestoreBatteryMode("2"))
			require.NoError(t, restore.RestoreBatteryMode(""))
			require.Error(t, restore.RestoreBatteryMode("1"))
			require.Error(t, restore.RestoreBatteryMode("invalid"))
			defaultMode := "2"
			if fallback == "time-of-use" {
				defaultMode = "10"
			}
			require.Equal(t, []string{"10", "2", defaultMode}, modes)
		})
	}
}
