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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
}
