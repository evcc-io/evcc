package tesla

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandResponse(t *testing.T) {
	sponsor.Subject = "any"

	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/1/vehicles/abc/command/charge_start":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"response": null, "error": "vehicle unavailable: vehicle is offline or asleep"}`))
		}
	}))
	srv.Start()

	client := NewAPI(util.NewLogger("foo"), srv.Client(), srv.URL)

	require.ErrorIs(t, NewController(client, "abc").ChargeEnable(true), api.ErrAsleep)
}

func TestVehicleDataRequestsLocation(t *testing.T) {
	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Query().Get("endpoints"), "location_data")
		w.Write([]byte(`{"response": {"drive_state": {"latitude": 1.5, "longitude": 2.5}}}`))
	}))
	srv.Start()

	res, err := NewAPI(util.NewLogger("foo"), srv.Client(), srv.URL).VehicleData("abc")
	require.NoError(t, err)
	require.Equal(t, 1.5, res.Response.DriveState.Latitude)
}

func TestVehicleDataWithoutLocationScope(t *testing.T) {
	var calls int
	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if strings.Contains(r.URL.Query().Get("endpoints"), "location_data") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Write([]byte(`{"response": {"charge_state": {"usable_battery_level": 42}}}`))
	}))
	srv.Start()

	client := NewAPI(util.NewLogger("foo"), srv.Client(), srv.URL)

	res, err := client.VehicleData("abc")
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.EqualValues(t, 42, res.Response.ChargeState.UsableBatteryLevel)

	_, err = client.VehicleData("abc")
	require.NoError(t, err)
	require.Equal(t, 3, calls)
}
