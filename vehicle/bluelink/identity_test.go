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
// EU, Hyundai AU): Login must always use the unmodified legacy refreshToken
// path, regardless of whether the password happens to look like a legacy
// token or a real account password.
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

// TestLoginUsesLegacyWhenPasswordLooksLikeLegacyToken covers a CCI-capable
// brand (Kia/Hyundai EU) where the configured password still has the shape
// of a legacy refresh_token: this must keep using the unmodified legacy
// path, not the CCI one, for users who still hold a valid pre-block token.
// The CCI endpoint is deliberately left unreachable - if the code mistakenly
// routed there, this test would fail with a connection error instead of the
// expected legacy access token.
func TestLoginUsesLegacyWhenPasswordLooksLikeLegacyToken(t *testing.T) {
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

// TestLoginUsesCCIAndWiresRefreshCCI covers a CCI-capable brand with a real
// account password: Login must perform the full CCI login and wire
// TokenSource to refreshCCI (not the legacy refreshToken) for subsequent
// refreshes. The mock CCS exchange deliberately returns a token that expires
// in 5s, which lands inside oauth2's default 10s expiry buffer - so the very
// next Token() call is guaranteed to trigger exactly one refresh, and
// asserting it hit the CCI refresh endpoint (rather than the legacy one, or
// none at all) confirms the wiring without needing internal hooks.
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

	var refreshCalls int32
	var exchangeCalls int32
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
		// The first (login) exchange deliberately returns a token that
		// expires in 5s - inside oauth2's default 10s expiry buffer, so it's
		// already considered invalid by the time the test asks for it. Every
		// later (post-refresh) exchange returns a long-lived token instead,
		// so refreshing settles after exactly one round-trip.
		resp := map[string]any{"accessToken": "ccs-token-2", "expiresTime": time.Now().Add(time.Hour).UnixMilli()}
		if atomic.AddInt32(&exchangeCalls, 1) == 1 {
			resp["accessToken"] = "ccs-token-1"
			resp["expiresTime"] = time.Now().Add(5 * time.Second).UnixMilli()
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	cciMux.HandleFunc("/domain/api/v2/auth/token-refresh", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&refreshCalls, 1)
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

	// The freshly issued token is already inside the expiry buffer, so this
	// very first Token() call is guaranteed to trigger exactly one refresh.
	// Asserting it hit the CCI refresh endpoint (rather than the legacy one,
	// or none at all) confirms TokenSource is wired to refreshCCI.
	token, err := identity.Token()
	require.NoError(t, err)
	assert.Equal(t, "ccs-token-2", token.AccessToken)
	assert.EqualValues(t, 1, atomic.LoadInt32(&refreshCalls), "expected TokenSource to be wired to refreshCCI")

	// the now long-lived token must not trigger a further refresh
	_, err = identity.Token()
	require.NoError(t, err)
	assert.EqualValues(t, 1, atomic.LoadInt32(&refreshCalls), "unexpected additional refresh")
}

// TestLoginUsesCCIBundleFromPasswordField covers the cci1: compound-bundle
// password: Login must use it directly via the CCI path without contacting
// the login or CCI servers at all (both are left unreachable here; a valid,
// non-expired bundle needs no network calls).
func TestLoginUsesCCIBundleFromPasswordField(t *testing.T) {
	deviceSrv := httptest.NewServer(http.HandlerFunc(deviceIDHandler))
	defer deviceSrv.Close()

	identity := newLoginTestIdentity(t, unreachable, deviceSrv.URL, unreachable, testCCIConfig())

	valid := cciBundle{
		AccessToken:  "bundle-ccs-token",
		RefreshToken: "bundle-refresh",
		Expiry:       time.Now().Add(time.Hour),
		DeviceID:     "device-bundle",
	}
	password := encodeCCIBundleForTest(t, valid)

	require.NoError(t, identity.Login("user@example.com", password, "en", "kia"))

	token, err := identity.Token()
	require.NoError(t, err)
	assert.Equal(t, "bundle-ccs-token", token.AccessToken)
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

// TestLoginPropagatesCCIError covers the CCI path: a failure there (here, the
// WAF-block detection from step 1) must also surface prefixed with
// "login failed: ", the same convention as the legacy path.
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
