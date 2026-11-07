package bluelink

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestChargeControlURI(t *testing.T) {
	for _, tc := range []struct {
		name, base, ccs2, legacy string
	}{
		{
			name:   "au",
			base:   "https://au-apigw.ccs.hyundai.com.au:8080/api/v1/spa",
			ccs2:   "https://au-apigw.ccs.hyundai.com.au:8080/api/v2/spa/vehicles/VID/ccs2/control/charge",
			legacy: "https://au-apigw.ccs.hyundai.com.au:8080/api/v1/spa/vehicles/VID/control/charge",
		},
		{
			name:   "eu",
			base:   "https://prd.eu-ccapi.hyundai.com:8080/api/v1/spa",
			ccs2:   "https://prd.eu-ccapi.hyundai.com:8080/api/v2/spa/vehicles/VID/ccs2/control/charge",
			legacy: "https://prd.eu-ccapi.hyundai.com:8080/api/v1/spa/vehicles/VID/control/charge",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.ccs2, chargeControlURI(tc.base, "VID", true))
			assert.Equal(t, tc.legacy, chargeControlURI(tc.base, "VID", false))
		})
	}
}

func TestDecodeChargeControl(t *testing.T) {
	require.NoError(t, decodeChargeControl(nil))
	require.NoError(t, decodeChargeControl([]byte("  \n")))
	require.NoError(t, decodeChargeControl([]byte(`{"retCode":"S"}`)))
	require.Error(t, decodeChargeControl([]byte(`{}`)))
	require.Error(t, decodeChargeControl([]byte(`{"msgId":"abc"}`)))
	require.Error(t, decodeChargeControl([]byte(`null`)))
	require.Error(t, decodeChargeControl([]byte(`{"retCode":"F"}`)))
}

func TestPinResponseResMsg(t *testing.T) {
	var res pinResponse
	require.NoError(t, json.Unmarshal([]byte(`{"retCode":"S","resMsg":{"controlToken":"tok","expiresTime":1700000000}}`), &res))
	assert.Equal(t, "tok", res.ResMsg.ControlToken)
	assert.EqualValues(t, 1700000000, res.ResMsg.ExpiresTime)

	var top pinResponse
	require.NoError(t, json.Unmarshal([]byte(`{"controlToken":"legacy","expiresTime":1700000100}`), &top))
	assert.Empty(t, top.ResMsg.ControlToken)
	assert.Equal(t, "legacy", top.ControlToken)
}

func TestControlTokenExpiry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	assert.Equal(t, time.Unix(1_700_000_600, 0).Add(-30*time.Second), controlTokenExpiry(1_700_000_600, now))

	past := controlTokenExpiry(1_699_999_000, now)
	assert.True(t, past.Before(now), "absolute expiresTime in the past must not keep the token cached")
}

func TestChargeStartStop(t *testing.T) {
	assert.Equal(t, "start", chargeStartStop(true))
	assert.Equal(t, "stop", chargeStartStop(false))
}

func TestProviderIsNotChargeController(t *testing.T) {
	p := NewProvider(NewAPI(util.NewLogger("test"), "https://example.invalid/api/v1/spa", func(*http.Request) error { return nil }), NewIdentity(util.NewLogger("test"), Config{}), Vehicle{}, "", time.Second)
	_, ok := api.Cap[api.ChargeController](p)
	assert.False(t, ok)
}

func TestChargeEnableCCS2UsesControlTokenHeaders(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/user/pin", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"retCode": "S",
			"resMsg": map[string]any{
				"controlToken": "ctrl",
				"expiresTime":  time.Now().Add(time.Hour).Unix(),
			},
		})
	})
	mux.HandleFunc("/api/v2/spa/vehicles/VID/ccs2/control/charge", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer ctrl", r.Header.Get("Authorization"))
		assert.Equal(t, "Bearer ctrl", r.Header.Get("AuthorizationCCSP"))
		assert.Equal(t, "1", r.Header.Get("Ccuccs2protocolsupport"))
		_ = json.NewEncoder(w).Encode(map[string]any{"retCode": "S"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	identity := NewIdentity(util.NewLogger("test"), Config{
		URI:               srv.URL,
		CCSPApplicationID: testCCSPApplicationID,
		Cfb:               testCfb,
	})
	identity.TokenSource = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "oauth-access"})
	identity.deviceID = "dev-1"

	client := NewAPI(util.NewLogger("test"), srv.URL+"/api/v1/spa", identity.Request)
	require.NoError(t, client.ChargeEnable(identity, Vehicle{VehicleID: "VID", CcuCCS2ProtocolSupport: 1}, "1234", false))
}

func TestChargeEnableCCS2RequiresPIN(t *testing.T) {
	client := NewAPI(util.NewLogger("test"), "https://example.invalid/api/v1/spa", func(*http.Request) error { return nil })
	err := client.ChargeEnable(NewIdentity(util.NewLogger("test"), Config{}), Vehicle{VehicleID: "VID", CcuCCS2ProtocolSupport: 1}, "", true)
	require.ErrorIs(t, err, api.ErrMissingCredentials)
}

func TestChargeEnableCCS2UnauthorizedInvalidatesControlToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/user/pin", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"retCode": "S",
			"resMsg": map[string]any{
				"controlToken": "ctrl",
				"expiresTime":  time.Now().Add(time.Hour).Unix(),
			},
		})
	})
	mux.HandleFunc("/api/v2/spa/vehicles/VID/ccs2/control/charge", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	mux.HandleFunc("/api/v1/spa/vehicles/VID/control/charge", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"retCode": "S"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	identity := NewIdentity(util.NewLogger("test"), Config{
		URI:               srv.URL,
		CCSPApplicationID: testCCSPApplicationID,
		Cfb:               testCfb,
	})
	identity.TokenSource = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "oauth-access"})
	identity.deviceID = "dev-1"
	identity.controlToken = "Bearer stale"
	identity.controlTokenExpiry = time.Now().Add(time.Hour)

	client := NewAPI(util.NewLogger("test"), srv.URL+"/api/v1/spa", identity.Request)
	require.NoError(t, client.ChargeEnable(identity, Vehicle{VehicleID: "VID", CcuCCS2ProtocolSupport: 1}, "1234", true))

	identity.mu.Lock()
	defer identity.mu.Unlock()
	assert.Empty(t, identity.controlToken)
	assert.True(t, identity.controlTokenExpiry.IsZero())
}

func TestChargeEnableCCS2FallsBackToLegacy(t *testing.T) {
	var ccs2, legacy int
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/user/pin", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"retCode": "S",
			"resMsg": map[string]any{
				"controlToken": "ctrl",
				"expiresTime":  time.Now().Add(time.Hour).Unix(),
			},
		})
	})
	mux.HandleFunc("/api/v2/spa/vehicles/VID/ccs2/control/charge", func(w http.ResponseWriter, _ *http.Request) {
		ccs2++
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("/api/v1/spa/vehicles/VID/control/charge", func(w http.ResponseWriter, _ *http.Request) {
		legacy++
		_ = json.NewEncoder(w).Encode(map[string]any{"retCode": "S"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	identity := NewIdentity(util.NewLogger("test"), Config{
		URI:               srv.URL,
		CCSPApplicationID: testCCSPApplicationID,
		Cfb:               testCfb,
	})
	identity.TokenSource = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "oauth-access"})
	identity.deviceID = "dev-1"

	client := NewAPI(util.NewLogger("test"), srv.URL+"/api/v1/spa", identity.Request)
	require.NoError(t, client.ChargeEnable(identity, Vehicle{VehicleID: "VID", CcuCCS2ProtocolSupport: 1}, "1234", false))
	assert.Equal(t, 1, ccs2)
	assert.Equal(t, 1, legacy)
}

func TestChargeEnableJoinsCCS2AndLegacyErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/user/pin", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"retCode": "S",
			"resMsg": map[string]any{
				"controlToken": "ctrl",
				"expiresTime":  time.Now().Add(time.Hour).Unix(),
			},
		})
	})
	mux.HandleFunc("/api/v2/spa/vehicles/VID/ccs2/control/charge", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	mux.HandleFunc("/api/v1/spa/vehicles/VID/control/charge", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	identity := NewIdentity(util.NewLogger("test"), Config{
		URI:               srv.URL,
		CCSPApplicationID: testCCSPApplicationID,
		Cfb:               testCfb,
	})
	identity.TokenSource = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "oauth-access"})
	identity.deviceID = "dev-1"

	client := NewAPI(util.NewLogger("test"), srv.URL+"/api/v1/spa", identity.Request)
	err := client.ChargeEnable(identity, Vehicle{VehicleID: "VID", CcuCCS2ProtocolSupport: 1}, "1234", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "502")
	assert.Contains(t, err.Error(), "503")
}
