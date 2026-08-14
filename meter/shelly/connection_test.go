package shelly

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shellyServer emulates a gen2+ device serving the given rpc responses. Methods
// not listed are reported as unavailable by Shelly.ListMethods and answered 404.
func shellyServer(t *testing.T, rpc map[string]string) *httptest.Server {
	t.Helper()

	methods := make([]string, 0, len(rpc))
	for m := range rpc {
		methods = append(methods, m)
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/shelly":
			json.NewEncoder(w).Encode(map[string]any{"gen": 2, "model": "SNSW-001X16EU"})

		case "/rpc/Shelly.ListMethods":
			json.NewEncoder(w).Encode(Gen2Methods{Methods: methods})

		default:
			method := strings.TrimPrefix(r.URL.Path, "/rpc/")
			res, ok := rpc[method]
			if !ok {
				http.Error(w, `{"code":-105,"message":"no handler"}`, http.StatusNotFound)
				return
			}
			w.Write([]byte(res))
		}
	}))
}

// TestNakedSwitchConnection asserts that a switch without power measurement
// (Shelly Plus 1) connects - its status has neither aenergy nor ret_aenergy.
func TestNakedSwitchConnection(t *testing.T) {
	srv := shellyServer(t, map[string]string{
		"Switch.GetStatus": `{"id":0,"source":"init","output":true,"temperature":{"tC":45.2,"tF":113.4}}`,
		"Switch.GetConfig": `{"id":0,"name":null,"in_mode":"follow","initial_state":"match_input","auto_on":false}`,
	})
	defer srv.Close()

	conn, err := NewConnection(srv.URL, "", "", 0, time.Second)
	require.NoError(t, err, "naked switch must connect")

	assert.False(t, conn.IsReversed())
	assert.False(t, conn.HasReturnEnergy())

	enabled, err := conn.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	total, err := conn.TotalEnergy()
	require.NoError(t, err)
	assert.Zero(t, total)

	ret, err := conn.ReturnEnergy()
	require.NoError(t, err)
	assert.Zero(t, ret)
}

// TestSwitchConnectionStatusError asserts that a temporarily unavailable switch
// status does not break connecting- the error surfaces on read instead.
func TestSwitchConnectionStatusError(t *testing.T) {
	// Switch.GetStatus advertised but erroring, e.g. component busy right after boot
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/shelly":
			json.NewEncoder(w).Encode(map[string]any{"gen": 2, "model": "SNSW-001X16EU"})
		case "/rpc/Shelly.ListMethods":
			json.NewEncoder(w).Encode(Gen2Methods{Methods: []string{"Switch.GetStatus", "Switch.GetConfig"}})
		case "/rpc/Switch.GetConfig":
			w.Write([]byte(`{"id":0}`))
		default:
			http.Error(w, `{"code":-114,"message":"component not found"}`, http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn, err := NewConnection(srv.URL, "", "", 0, time.Second)
	require.NoError(t, err, "status error must not break connecting")

	_, err = conn.CurrentPower()
	require.Error(t, err, "error surfaces on read")
}

// TestPlugConnection covers the metered plug from #32213: aenergy without ret_aenergy.
func TestPlugConnection(t *testing.T) {
	srv := shellyServer(t, map[string]string{
		"Switch.GetStatus": `{"id":0,"output":true,"apower":399.2,"voltage":240.8,"current":1.886,"aenergy":{"total":2574466.629}}`,
		"Switch.GetConfig": `{"id":0,"power_limit":2800,"voltage_limit":280}`,
	})
	defer srv.Close()

	conn, err := NewConnection(srv.URL, "", "", 0, time.Second)
	require.NoError(t, err)

	assert.False(t, conn.IsReversed())
	assert.False(t, conn.HasReturnEnergy(), "plug has no return register")

	total, err := conn.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 2574.466629, total)
}

// TestReversedSwitchConnection covers a switch with device-side reverse measurement.
func TestReversedSwitchConnection(t *testing.T) {
	srv := shellyServer(t, map[string]string{
		"Switch.GetStatus": `{"id":0,"output":true,"apower":-350,"aenergy":{"total":10000},"ret_aenergy":{"total":4000}}`,
		"Switch.GetConfig": `{"id":0,"reverse":true}`,
	})
	defer srv.Close()

	conn, err := NewConnection(srv.URL, "", "", 0, time.Second)
	require.NoError(t, err)

	assert.True(t, conn.IsReversed())
	assert.True(t, conn.HasReturnEnergy())

	total, err := conn.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 6.0, total)

	ret, err := conn.ReturnEnergy()
	require.NoError(t, err)
	assert.Equal(t, 4.0, ret)
}
