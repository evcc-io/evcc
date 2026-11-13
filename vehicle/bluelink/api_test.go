package bluelink

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
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

func testIdentity(t *testing.T, uri string) *Identity {
	t.Helper()
	identity := NewIdentity(util.NewLogger("test"), Config{
		URI:               uri,
		CCSPApplicationID: testCCSPApplicationID,
		Cfb:               testCfb,
	})
	identity.TokenSource = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "oauth-access"})
	identity.deviceID = "dev-1"
	return identity
}

func TestChargeEnableCCS2(t *testing.T) {
	var pin, charge int
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/user/pin", func(w http.ResponseWriter, r *http.Request) {
		pin++
		assert.Equal(t, http.MethodPut, r.Method)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"resMsg": map[string]any{"controlToken": "ctrl"},
		})
	})
	mux.HandleFunc("/api/v2/spa/vehicles/VID/ccs2/control/charge", func(w http.ResponseWriter, r *http.Request) {
		charge++
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer ctrl", r.Header.Get("Authorization"))
		assert.Equal(t, "Bearer ctrl", r.Header.Get("AuthorizationCCSP"))
		assert.Equal(t, "1", r.Header.Get("Ccuccs2protocolsupport"))
		_ = json.NewEncoder(w).Encode(map[string]any{"retCode": "S"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	identity := testIdentity(t, srv.URL)
	client := NewAPI(util.NewLogger("test"), srv.URL+"/api/v1/spa", identity.Request)
	require.NoError(t, client.ChargeEnable(identity, Vehicle{VehicleID: "VID", CcuCCS2ProtocolSupport: 1}, "1234", false))
	assert.Equal(t, 1, pin)
	assert.Equal(t, 1, charge)
}

func TestChargeEnableLegacy(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/user/pin", func(http.ResponseWriter, *http.Request) {
		t.Fatal("legacy charge control must not verify PIN")
	})
	mux.HandleFunc("/api/v1/spa/vehicles/VID/control/charge", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		var data map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&data))
		assert.Equal(t, "start", data["action"])
		assert.Equal(t, "dev-1", data["deviceId"])
		_ = json.NewEncoder(w).Encode(map[string]any{"retCode": "S"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	identity := testIdentity(t, srv.URL)
	client := NewAPI(util.NewLogger("test"), srv.URL+"/api/v1/spa", identity.Request)
	require.NoError(t, client.ChargeEnable(identity, Vehicle{VehicleID: "VID"}, "", true))
}
