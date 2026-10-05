package bluelink

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unreachable is a host nobody listens on, used to prove that a code path
// does not perform any HTTP call (fast-failing rather than a slow DNS lookup).
const unreachable = "http://127.0.0.1:1"

// newTestIdentity creates an Identity for the given test, using the test name
// as the settings key discriminator (identity.user) so that parallel/repeated
// test runs never share a db/settings entry with one another.
func newTestIdentity(t *testing.T, loginURL, cciURL string) *Identity {
	t.Helper()

	config := Config{
		// Brand carries the test name so each (sub)test gets its own
		// db/settings key (see settingsKey), without changing the
		// username actually sent in login requests.
		Brand:         "kia-test:" + t.Name(),
		LoginFormHost: loginURL,
		CCI: &CCIConfig{
			OneAppClientID:       "test-client-id",
			OneAppRedirectURI:    "https://oneapp.kia.com/redirect",
			APIURL:               cciURL,
			PackageID:            "com.kia.oneapp.eu",
			ClientName:           "kia",
			OSVersion:            "27",
			NotificationProvider: "IOS_APPSTORE",
		},
	}

	identity := NewIdentity(util.NewLogger("test"), config)
	identity.user = "test@example.com"
	identity.language = "en"

	// db/settings.Delete requires an initialised gorm DB, which these
	// unit tests don't set up. SetString only ever touches the in-memory
	// cache (see db/settings/setting.go), so blanking the key this way
	// is enough to keep tests isolated without needing a real database.
	t.Cleanup(func() { settings.SetString(identity.settingsKey(), "") })

	return identity
}

func jwkParam(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("<html>login</html>"))
}

func certsHandler(priv *rsa.PrivateKey) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		n := jwkParam(priv.PublicKey.N.Bytes())
		e := jwkParam(big.NewInt(int64(priv.PublicKey.E)).Bytes())
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"retValue":{"kid":"test-kid","n":%q,"e":%q}}`, n, e)
	}
}

// TestParseCCSExpiry is a regression test for a bug found against the real Kia
// backend: an expiresTime read in the wrong unit makes every token look expired
func TestParseCCSExpiry(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		expiresTime int64
		wantWithin  time.Duration // expected to be within [now, now+wantWithin]
	}{
		{"zero falls back to default", 0, ccsExpiryFallback + time.Minute},
		{"negative falls back to default", -1, ccsExpiryFallback + time.Minute},
		{"second epoch", now.Add(time.Hour).Unix(), 2 * time.Hour},
		{"millisecond epoch falls back to default", now.Add(time.Hour).UnixMilli(), ccsExpiryFallback + time.Minute},
		{"implausible value falls back to default", 123, ccsExpiryFallback + time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCCSExpiry(tt.expiresTime)
			assert.True(t, got.After(now), "expiry must be in the future, got %v", got)
			assert.True(t, got.Before(now.Add(tt.wantWithin)), "expiry too far out, got %v", got)
		})
	}
}

func TestLoginCCIPasswordSuccess(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	const password = "s3cret-Passw0rd!"

	loginMux := http.NewServeMux()
	loginMux.HandleFunc("/auth/api/v2/user/oauth2/authorize", okHandler)
	loginMux.HandleFunc("/auth/api/v1/accounts/certs", certsHandler(priv))
	loginMux.HandleFunc("/auth/account/signin", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())

		ciphertext, err := hex.DecodeString(r.FormValue("password"))
		require.NoError(t, err)
		plain, err := rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
		require.NoError(t, err)

		assert.Equal(t, password, string(plain))
		assert.Equal(t, "true", r.FormValue("encryptedPassword"))
		assert.Equal(t, "test@example.com", r.FormValue("username"))

		w.Header().Set("Location", "https://oneapp.kia.com/redirect?code=testcode&state=ccsp")
		w.WriteHeader(http.StatusFound)
	})
	loginSrv := httptest.NewServer(loginMux)
	defer loginSrv.Close()

	cciMux := http.NewServeMux()
	cciMux.HandleFunc("/domain/api/v1/auth/token", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "testcode", r.URL.Query().Get("code"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accessToken":              "cci-access-1",
			"refreshToken":             "cci-refresh-1",
			"nonCcsToken":              "non-ccs-1",
			"exchangeableAccessToken":  "exch-access-1",
			"exchangeableRefreshToken": "exch-refresh-1",
			"nonCcsRefreshToken":       "non-ccs-refresh-1",
			"idToken":                  "id-1",
			"expiresIn":                3599,
		})
	})
	cciMux.HandleFunc("/domain/api/v1/auth/token-exchange", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "CCS", r.URL.Query().Get("serviceType"))
		assert.Equal(t, "Bearer cci-access-1", r.Header.Get("authorization"))
		assert.Equal(t, "non-ccs-1", r.Header.Get("Authentication"))
		assert.Equal(t, "exch-access-1", r.Header.Get("exchangeable-token"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accessToken": "ccs-token-1",
			"expiresTime": time.Now().Add(time.Hour).Unix(),
		})
	})
	cciSrv := httptest.NewServer(cciMux)
	defer cciSrv.Close()

	identity := newTestIdentity(t, loginSrv.URL, cciSrv.URL)

	token, err := identity.loginCCIPassword(password)
	require.NoError(t, err)

	assert.Equal(t, "ccs-token-1", token.AccessToken)
	assert.Equal(t, "cci-refresh-1", token.RefreshToken)
	assert.True(t, token.Valid())
	assert.Equal(t, "cci-access-1", identity.bundle.CCIAccessToken)
	assert.Equal(t, "exch-access-1", identity.bundle.ExchangeableToken)
	assert.Equal(t, "non-ccs-1", identity.bundle.NonCcsToken)
	assert.NotEmpty(t, identity.bundle.DeviceID)

	// persisted so a restart can reuse/refresh it instead of logging in again
	var persisted cciBundle
	require.NoError(t, settings.Json(identity.settingsKey(), &persisted))
	assert.Equal(t, "ccs-token-1", persisted.AccessToken)
	assert.Equal(t, "cci-refresh-1", persisted.RefreshToken)
}

func TestLoginCCIPasswordWAFBlocked(t *testing.T) {
	loginMux := http.NewServeMux()
	loginMux.HandleFunc("/auth/api/v2/user/oauth2/authorize", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Your request looks like abusing our system"))
	})
	loginSrv := httptest.NewServer(loginMux)
	defer loginSrv.Close()

	identity := newTestIdentity(t, loginSrv.URL, unreachable)

	_, err := identity.loginCCIPassword("whatever")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAF")
}

func TestLoginCCIPasswordSigninRejected(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	loginMux := http.NewServeMux()
	loginMux.HandleFunc("/auth/api/v2/user/oauth2/authorize", okHandler)
	loginMux.HandleFunc("/auth/api/v1/accounts/certs", certsHandler(priv))
	loginMux.HandleFunc("/auth/account/signin", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("invalid credentials"))
	})
	loginSrv := httptest.NewServer(loginMux)
	defer loginSrv.Close()

	identity := newTestIdentity(t, loginSrv.URL, unreachable)

	_, err = identity.loginCCIPassword("wrong-password")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signin failed")
}

func TestLoginCCIPasswordConsentRequired(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	loginMux := http.NewServeMux()
	loginMux.HandleFunc("/auth/api/v2/user/oauth2/authorize", okHandler)
	loginMux.HandleFunc("/auth/api/v1/accounts/certs", certsHandler(priv))
	loginMux.HandleFunc("/auth/account/signin", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "https://idpconnect-eu.kia.com/web/v1/user/authorization?foo=bar")
		w.WriteHeader(http.StatusFound)
	})
	loginSrv := httptest.NewServer(loginMux)
	defer loginSrv.Close()

	identity := newTestIdentity(t, loginSrv.URL, unreachable)

	_, err = identity.loginCCIPassword("whatever")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "consent")
}

func TestRefreshCCI(t *testing.T) {
	cciMux := http.NewServeMux()
	cciMux.HandleFunc("/domain/api/v2/auth/token-refresh", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "old-cci-refresh", body["refreshToken"])

		_ = json.NewEncoder(w).Encode(map[string]any{
			"accessToken":              "cci-access-2",
			"refreshToken":             "cci-refresh-2",
			"nonCcsToken":              "non-ccs-2",
			"exchangeableAccessToken":  "exch-access-2",
			"exchangeableRefreshToken": "exch-refresh-2",
			"nonCcsRefreshToken":       "non-ccs-refresh-2",
			"idToken":                  "id-2",
		})
	})
	cciMux.HandleFunc("/domain/api/v1/auth/token-exchange", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer cci-access-2", r.Header.Get("authorization"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accessToken": "ccs-token-2",
			"expiresTime": time.Now().Add(time.Hour).Unix(),
		})
	})
	cciSrv := httptest.NewServer(cciMux)
	defer cciSrv.Close()

	identity := newTestIdentity(t, unreachable, cciSrv.URL)

	old := cciBundle{
		AccessToken:              "ccs-token-1",
		RefreshToken:             "old-cci-refresh",
		Expiry:                   time.Now().Add(-time.Hour), // expired
		DeviceID:                 "device-abc",
		CCIAccessToken:           "old-cci-access",
		ExchangeableToken:        "old-exch",
		ExchangeableRefreshToken: "old-exch-refresh",
		NonCcsToken:              "old-non-ccs",
		NonCcsRefreshToken:       "old-non-ccs-refresh",
		IDToken:                  "old-id",
	}

	identity.bundle = old

	newToken, err := identity.refreshCCI(nil)
	require.NoError(t, err)
	assert.Equal(t, "ccs-token-2", newToken.AccessToken)
	assert.Equal(t, "cci-refresh-2", newToken.RefreshToken)
	assert.True(t, newToken.Valid())

	var persisted cciBundle
	require.NoError(t, settings.Json(identity.settingsKey(), &persisted))
	assert.Equal(t, "ccs-token-2", persisted.AccessToken)
	assert.Equal(t, "device-abc", persisted.DeviceID) // device id carried over unchanged
}

func TestLoginCCIUsesPersistedBundleWithoutContactingServer(t *testing.T) {
	identity := newTestIdentity(t, unreachable, unreachable)

	valid := cciBundle{
		AccessToken:  "still-valid-ccs",
		RefreshToken: "still-valid-refresh",
		Expiry:       time.Now().Add(time.Hour),
		DeviceID:     "device-xyz",
	}
	require.NoError(t, settings.SetJson(identity.settingsKey(), valid))

	token, err := identity.loginCCI("irrelevant-password-value")
	require.NoError(t, err)
	assert.Equal(t, "still-valid-ccs", token.AccessToken)
}

// TestLoginCCIFallsBackToPasswordLoginWhenPersistedBundleUnusable covers a
// persisted bundle that is expired and unrefreshable: loginCCI must not get
// stuck on that stale state but fall back to the full password login
func TestLoginCCIFallsBackToPasswordLoginWhenPersistedBundleUnusable(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	const password = "s3cret-Passw0rd!"

	loginMux := http.NewServeMux()
	loginMux.HandleFunc("/auth/api/v2/user/oauth2/authorize", okHandler)
	loginMux.HandleFunc("/auth/api/v1/accounts/certs", certsHandler(priv))
	loginMux.HandleFunc("/auth/account/signin", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		ciphertext, err := hex.DecodeString(r.FormValue("password"))
		require.NoError(t, err)
		plain, err := rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
		require.NoError(t, err)
		assert.Equal(t, password, string(plain))

		w.Header().Set("Location", "https://oneapp.kia.com/redirect?code=testcode&state=ccsp")
		w.WriteHeader(http.StatusFound)
	})
	loginSrv := httptest.NewServer(loginMux)
	defer loginSrv.Close()

	cciMux := http.NewServeMux()
	cciMux.HandleFunc("/domain/api/v1/auth/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accessToken": "fresh-cci-access", "refreshToken": "fresh-cci-refresh",
			"nonCcsToken": "fresh-non-ccs", "exchangeableAccessToken": "fresh-exch-access",
			"exchangeableRefreshToken": "fresh-exch-refresh", "nonCcsRefreshToken": "fresh-non-ccs-refresh",
			"idToken": "fresh-id", "expiresIn": 3599,
		})
	})
	cciMux.HandleFunc("/domain/api/v1/auth/token-exchange", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accessToken": "fresh-ccs-token",
			"expiresTime": time.Now().Add(time.Hour).Unix(),
		})
	})
	cciSrv := httptest.NewServer(cciMux)
	defer cciSrv.Close()

	identity := newTestIdentity(t, loginSrv.URL, cciSrv.URL)

	// stale/corrupted settings state: expired and no refresh token
	stale := cciBundle{
		AccessToken: "stale-ccs-token",
		Expiry:      time.Now().Add(-time.Hour),
		DeviceID:    "stale-device-id",
	}
	require.NoError(t, settings.SetJson(identity.settingsKey(), stale))

	token, err := identity.loginCCI(password)
	require.NoError(t, err)
	assert.Equal(t, "fresh-ccs-token", token.AccessToken)
	assert.NotEqual(t, "stale-ccs-token", token.AccessToken)

	var persisted cciBundle
	require.NoError(t, settings.Json(identity.settingsKey(), &persisted))
	assert.Equal(t, "fresh-ccs-token", persisted.AccessToken)
	assert.NotEqual(t, "stale-device-id", persisted.DeviceID, "a fresh login must not carry over the stale bundle's device id")
}
