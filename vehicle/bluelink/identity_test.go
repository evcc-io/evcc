package bluelink

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCCSPApplicationID/testCfb are copied from the real production Kia EU
// config (vehicle/bluelink.go) so Identity.stamp() succeeds during Login's
// getDeviceID call - not secrets, both are already public there.
const (
	testCCSPApplicationID = "a2b8469b-30a3-4361-8e13-6fceea8fbe74"
	testCfb               = "wLTVxwidmH8CfJYBWSnHD6E0huk0ozdiuygB4hLkM5XCgzAL1Dk5sE36d/bx5PFMbZs="
)

// deviceIDHandler serves Identity.getDeviceID's device registration endpoint,
// which Login always calls after either auth path succeeds.
func deviceIDHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"RetCode":"S","ResMsg":{"DeviceID":"test-device-id"}}`)
}

// legacyTokenHandler serves the legacy refresh_token grant endpoint used by
// Identity.refreshToken.
func legacyTokenHandler(accessToken, refreshToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":%q,"refresh_token":%q,"expires_in":3600}`, accessToken, refreshToken)
	}
}

// newLoginTestIdentity builds an Identity suitable for exercising the public
// Login() method end to end, including the legacy getDeviceID call it always
// makes. cci == nil mirrors a brand without CCI support (e.g. Genesis EU).
func newLoginTestIdentity(t *testing.T, loginURL, deviceURL, cciURL string, cci *CCIConfig) *Identity {
	t.Helper()

	config := Config{
		Brand:             "kia-login-test:" + t.Name(),
		URI:               deviceURL,
		CCSPServiceID:     "test-ccsp-service-id",
		CCSPServiceSecret: "test-ccsp-secret",
		CCSPApplicationID: testCCSPApplicationID,
		Cfb:               testCfb,
		LoginFormHost:     loginURL,
		PushType:          "APNS",
		CCI:               cci,
	}
	if cci != nil {
		config.CCI.APIURL = cciURL
	}

	return NewIdentity(util.NewLogger("test"), config)
}

func testCCIConfig() *CCIConfig {
	return &CCIConfig{
		OneAppClientID:       "test-client-id",
		OneAppRedirectURI:    "https://oneapp.kia.com/redirect",
		PackageID:            "com.kia.oneapp.eu",
		ClientName:           "kia",
		OSVersion:            "27",
		NotificationProvider: "IOS_APPSTORE",
	}
}

// TestLoginUsesLegacyWhenCCIIsNil covers brands without CCI support (Genesis
// EU, Hyundai AU): Login must always use the unmodified legacy refreshToken path
func TestLoginUsesLegacyWhenCCIIsNil(t *testing.T) {
	for _, tt := range []struct {
		name     string
		password string
	}{
		{"legacy-shaped password", strings.Repeat("A", 48)},
		{"real-looking password", "s3cret-Passw0rd!"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/auth/api/v2/user/oauth2/token", legacyTokenHandler("legacy-access", "legacy-refresh"))
			loginSrv := httptest.NewServer(mux)
			defer loginSrv.Close()

			deviceSrv := httptest.NewServer(http.HandlerFunc(deviceIDHandler))
			defer deviceSrv.Close()

			identity := newLoginTestIdentity(t, loginSrv.URL, deviceSrv.URL, "", nil)

			require.NoError(t, identity.Login("user@example.com", tt.password, "en", "kia"))

			token, err := identity.Token()
			require.NoError(t, err)
			assert.Equal(t, "legacy-access", token.AccessToken)
		})
	}
}

// TestLoginPrefersLegacyToken covers a CCI-capable brand whose configured
// password is still a valid legacy refresh_token: the CCI path must not be used
// (its endpoint is left unreachable, so routing there would fail the test)
func TestLoginPrefersLegacyToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/api/v2/user/oauth2/token", legacyTokenHandler("legacy-access-2", "legacy-refresh-2"))
	loginSrv := httptest.NewServer(mux)
	defer loginSrv.Close()

	deviceSrv := httptest.NewServer(http.HandlerFunc(deviceIDHandler))
	defer deviceSrv.Close()

	identity := newLoginTestIdentity(t, loginSrv.URL, deviceSrv.URL, unreachable, testCCIConfig())

	require.NoError(t, identity.Login("user@example.com", strings.Repeat("A", 48), "en", "kia"))

	token, err := identity.Token()
	require.NoError(t, err)
	assert.Equal(t, "legacy-access-2", token.AccessToken)
}

// TestLoginUsesCCIAndWiresRefreshCCI covers a CCI-capable brand with an account
// password: Login must fall back to the CCI login and wire TokenSource to refreshCCI
func TestLoginUsesCCIAndWiresRefreshCCI(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	loginMux := http.NewServeMux()
	loginMux.HandleFunc("/auth/api/v2/user/oauth2/authorize", okHandler)
	loginMux.HandleFunc("/auth/api/v1/accounts/certs", certsHandler(priv))
	loginMux.HandleFunc("/auth/account/signin", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		ciphertext, err := hex.DecodeString(r.FormValue("password"))
		require.NoError(t, err)
		_, err = rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
		require.NoError(t, err)

		w.Header().Set("Location", "https://oneapp.kia.com/redirect?code=testcode&state=ccsp")
		w.WriteHeader(http.StatusFound)
	})
	loginSrv := httptest.NewServer(loginMux)
	defer loginSrv.Close()

	var refreshCalls atomic.Int32
	var exchangeCalls atomic.Int32
	cciMux := http.NewServeMux()
	cciMux.HandleFunc("/domain/api/v1/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accessToken": "cci-access", "refreshToken": "cci-refresh",
			"nonCcsToken": "non-ccs", "exchangeableAccessToken": "exch-access",
			"exchangeableRefreshToken": "exch-refresh", "nonCcsRefreshToken": "non-ccs-refresh",
			"idToken": "id-1", "expiresIn": 3599,
		})
	})
	cciMux.HandleFunc("/domain/api/v1/auth/token-exchange", func(w http.ResponseWriter, _ *http.Request) {
		// the first (login) exchange returns a token expiring inside oauth2's
		// 10s buffer, so the next Token() call triggers exactly one refresh
		resp := map[string]any{"accessToken": "ccs-token-2", "expiresTime": time.Now().Add(time.Hour).Unix()}
		if exchangeCalls.Add(1) == 1 {
			resp["accessToken"] = "ccs-token-1"
			resp["expiresTime"] = time.Now().Add(5 * time.Second).Unix()
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	cciMux.HandleFunc("/domain/api/v2/auth/token-refresh", func(w http.ResponseWriter, _ *http.Request) {
		refreshCalls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accessToken": "cci-access-2", "refreshToken": "cci-refresh-2",
		})
	})
	cciSrv := httptest.NewServer(cciMux)
	defer cciSrv.Close()

	deviceSrv := httptest.NewServer(http.HandlerFunc(deviceIDHandler))
	defer deviceSrv.Close()

	identity := newLoginTestIdentity(t, loginSrv.URL, deviceSrv.URL, cciSrv.URL, testCCIConfig())

	require.NoError(t, identity.Login("user@example.com", "s3cret-Passw0rd!", "en", "kia"))

	// the freshly issued token is inside the expiry buffer, so this Token()
	// call must trigger exactly one refresh against the CCI endpoint
	token, err := identity.Token()
	require.NoError(t, err)
	assert.Equal(t, "ccs-token-2", token.AccessToken)
	assert.EqualValues(t, 1, refreshCalls.Load(), "expected TokenSource to be wired to refreshCCI")

	// the now long-lived token must not trigger a further refresh
	_, err = identity.Token()
	require.NoError(t, err)
	assert.EqualValues(t, 1, refreshCalls.Load(), "unexpected additional refresh")
}

// TestLoginPropagatesLegacyError covers the legacy path: a failure there must
// surface through Login prefixed with evcc's existing "login failed: " convention.
func TestLoginPropagatesLegacyError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/api/v2/user/oauth2/token", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	})
	loginSrv := httptest.NewServer(mux)
	defer loginSrv.Close()

	identity := newLoginTestIdentity(t, loginSrv.URL, unreachable, "", nil)

	err := identity.Login("user@example.com", strings.Repeat("A", 48), "en", "kia")
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "login failed:"), "got: %s", err.Error())
}

// TestLoginPropagatesCCIError covers the CCI path: a failure there must surface
// prefixed with "login failed: ", the same convention as the legacy path
func TestLoginPropagatesCCIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/api/v2/user/oauth2/authorize", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Your request looks like abusing our system"))
	})
	loginSrv := httptest.NewServer(mux)
	defer loginSrv.Close()

	identity := newLoginTestIdentity(t, loginSrv.URL, unreachable, unreachable, testCCIConfig())

	err := identity.Login("user@example.com", "s3cret-Passw0rd!", "en", "kia")
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "login failed:"), "got: %s", err.Error())
	assert.Contains(t, err.Error(), "WAF")
}
