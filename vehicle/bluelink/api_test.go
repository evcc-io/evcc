package bluelink

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

func TestStatusLatestFallsBackToCCS2On5031(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/ccs2/carstatus/latest") {
			fmt.Fprint(w, `{"retCode":"S","resCode":"0000","resMsg":{"state":{"Vehicle":{"Green":{"BatteryManagement":{"BatteryRemain":{"Ratio":42}}}}}}}`)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"retCode":"F","resCode":"5031","resMsg":"Unavailable remote control - Service Temporary Unavailable"}`)
	}))
	defer srv.Close()

	api := NewAPI(util.NewLogger("test"), srv.URL, func(*http.Request) error { return nil })

	res, err := api.StatusLatest(Vehicle{VehicleID: "vid"})
	require.NoError(t, err)

	soc, err := res.SoC()
	require.NoError(t, err)
	require.Equal(t, 42.0, soc)
	require.Equal(t, []string{"/api/v1/spa/vehicles/vid/status", "/api/v1/spa/vehicles/vid/status/latest", "/api/v1/spa/vehicles/vid/ccs2/carstatus/latest"}, paths)
}
