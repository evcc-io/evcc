package charger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/charger/nrg/connect"
	"github.com/stretchr/testify/require"
)

// https://github.com/evcc-io/evcc/issues/33625
func TestNRGKickConnectSinglePut(t *testing.T) {
	var puts []connect.Settings

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)

		var s connect.Settings
		require.NoError(t, json.NewDecoder(r.Body).Decode(&s))
		puts = append(puts, s)

		w.Write([]byte("{}"))
	}))
	defer srv.Close()

	nrg, err := NewNRGKickConnect(srv.URL, "mac", "pw", time.Second)
	require.NoError(t, err)

	// current while disabled is deferred to the enable request
	require.NoError(t, nrg.MaxCurrent(16))
	require.Empty(t, puts)

	require.NoError(t, nrg.Enable(true))
	require.Len(t, puts, 1)
	require.True(t, puts[0].Values.ChargingStatus.Charging)
	require.Equal(t, 16.0, puts[0].Values.ChargingCurrent.Value)

	// current while enabled is written immediately
	require.NoError(t, nrg.MaxCurrent(6))
	require.Len(t, puts, 2)
	require.True(t, puts[1].Values.ChargingStatus.Charging)
	require.Equal(t, 6.0, puts[1].Values.ChargingCurrent.Value)

	require.NoError(t, nrg.Enable(false))
	require.Len(t, puts, 3)
	require.False(t, puts[2].Values.ChargingStatus.Charging)
}
