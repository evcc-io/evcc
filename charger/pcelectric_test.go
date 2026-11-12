package charger

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPCElectricStatus(t *testing.T) {
	withSponsor(t)

	tests := []struct {
		name  string
		slave int
		body  string
		want  api.ChargeStatus // StatusNone means an error is expected
	}{
		// captured from a GLBDCM-T22EFC, firmware 1.3.8
		{
			name: "unplugged, always off",
			body: `{"connector":"NOT_CONNECTED","mode":"ALWAYS_OFF","powerMode":"OFF","chargeStatus":0,"sessionStartTime":null,"accSessionMillis":0,"sessionStartValue":-1,"pilotLevel":6,"currentChargingPower":0}`,
			want: api.StatusA,
		},
		{
			name: "plugged, always off, session cleared",
			body: `{"connector":"UNAVAILABLE","mode":"ALWAYS_OFF","chargeStatus":144,"sessionStartTime":null,"accSessionMillis":0,"pilotLevel":6}`,
			want: api.StatusB,
		},

		{name: "plugged, always off, session running", body: `{"chargeStatus":144,"sessionStartTime":1793205969221}`, want: api.StatusB},
		{name: "connected", body: `{"chargeStatus":48}`, want: api.StatusB},
		{name: "charging", body: `{"chargeStatus":64,"sessionStartTime":1793205969221}`, want: api.StatusC},
		{name: "cable fault", body: `{"chargeStatus":160}`, want: api.StatusNone},
		{name: "slave, session cleared", slave: 1, body: `[{"chargeStatus":0},{"chargeStatus":144,"sessionStartTime":null}]`, want: api.StatusB},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := "status"
			if tc.slave > 0 {
				path = "slaves/false"
			}

			mux := http.NewServeMux()
			mux.HandleFunc("/servlet/rest/chargebox/"+path, func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, tc.body)
			})

			srv := httptest.NewTestServer(t, mux)
			srv.Start()

			wb, err := NewPCElectric(srv.URL, tc.slave, "")
			require.NoError(t, err)

			status, err := wb.Status()
			if tc.want == api.StatusNone {
				assert.ErrorContains(t, err, "invalid status")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, status)
		})
	}
}
