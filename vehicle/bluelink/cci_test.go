package bluelink

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
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

	cci := testCCIConfig()
	cci.APIURL = cciURL

	config := Config{
		// Brand carries the test name so each (sub)test gets its own
		// db/settings key (see settingsKey), without changing the
		// username actually sent in login requests.
		Brand:         "kia-test:" + t.Name(),
		LoginFormHost: loginURL,
		CCI:           cci,
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

// assertSigned verifies the X-Fingerprint of a request against the test
// config's signing key and returns the request body
func assertSigned(t *testing.T, r *http.Request) []byte {
	t.Helper()

	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)

	headers := make(map[string]string)
	for _, k := range []string{"client-id", "client-name", "client-version", "client-os-code", "client-os-version", "X-Timestamp"} {
		headers[k] = r.Header.Get(k)
	}

	assert.Equal(t, "android", headers["client-os-code"])
	assert.Equal(t, "KiaEU", headers["client-name"])
	assert.Equal(t, fingerprint(headers, testCCIConfig().FingerprintKey, body), r.Header.Get("X-Fingerprint"))

	ts, err := strconv.ParseInt(headers["X-Timestamp"], 10, 64)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), time.UnixMilli(ts), time.Minute)

	return body
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

func TestFingerprint(t *testing.T) {
	identity := func(pkg, name, ts string) map[string]string {
		return map[string]string{
			"client-id":         pkg,
			"client-name":       name,
			"client-version":    "1.4.0",
			"client-os-code":    "android",
			"client-os-version": "14",
			"X-Timestamp":       ts,
		}
	}

	kia := identity("com.kia.oneapp.eu", "KiaEU", "1700000000123")
	kiaKey := "yb4Z1n2nlKe" + "ukNilE7QEdb" + "KB34nStPVa"
	hyundai := identity("com.hyundai.oneapp.eu", "HyundaiEU", "1792000084084")
	hyundaiKey := "OsZDfDuCyVX" + "yfqjCoA2JbX" + "2MrLpGwNBC"

	// golden vectors from hyundai_kia_connect_api, verified against the live gate
	assert.Equal(t, "0a3ceccc83823ec0d2c5d39a15dbf722c813e6da48800a0156a7a8459db63337", fingerprint(kia, kiaKey, nil))
	assert.Equal(t, "df04e3dadd52288ce5b280b368af261272556b5360adbc90a817a167526e2946", fingerprint(hyundai, hyundaiKey, nil))

	// body vector from an independent HMAC computation of the upstream recipe, body in Python json.dumps style
	assert.Equal(t, "0aae2e0acd5d57bcabba202cc7a839fc90f1d52814091b92e0908be66ed12eef", fingerprint(kia, kiaKey, []byte(`{"refreshToken": "r1", "idToken": "i1"}`)))

	bare := fingerprint(kia, kiaKey, nil)
	assert.Equal(t, bare, fingerprint(kia, kiaKey, []byte(" \t\r\n\v\f")), "ascii whitespace is not signed")
	assert.NotEqual(t, bare, fingerprint(kia, kiaKey, []byte("\u00a0")), "non-ascii whitespace is signed")
	assert.Equal(t, fingerprint(kia, kiaKey, []byte(`{"a":1}`)), fingerprint(kia, kiaKey, []byte(`{ "a": 1 }`)))
	assert.NotEqual(t, bare, fingerprint(kia, kiaKey, []byte(`{"a":1}`)), "body is signed")
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
	cciMux.HandleFunc("POST /auth/v1/token", func(w http.ResponseWriter, r *http.Request) {
		assertSigned(t, r)
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
	cciMux.HandleFunc("POST /auth/v1/token-exchange", func(w http.ResponseWriter, r *http.Request) {
		assertSigned(t, r)
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
	cciMux.HandleFunc("POST /auth/v2/token-refresh", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "device-abc", r.Header.Get("client-device-id"))
		body := assertSigned(t, r)

		var fields map[string]string
		require.NoError(t, json.Unmarshal(body, &fields))
		assert.Equal(t, map[string]string{
			"accessToken":              "old-cci-access",
			"refreshToken":             "old-cci-refresh",
			"exchangeableAccessToken":  "old-exch",
			"exchangeableRefreshToken": "old-exch-refresh",
			"nonCcsToken":              "old-non-ccs",
			"nonCcsRefreshToken":       "old-non-ccs-refresh",
			"idToken":                  "old-id",
		}, fields)

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
	cciMux.HandleFunc("POST /auth/v1/token-exchange", func(w http.ResponseWriter, r *http.Request) {
		assertSigned(t, r)
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
	cciMux.HandleFunc("POST /auth/v1/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accessToken": "fresh-cci-access", "refreshToken": "fresh-cci-refresh",
			"nonCcsToken": "fresh-non-ccs", "exchangeableAccessToken": "fresh-exch-access",
			"exchangeableRefreshToken": "fresh-exch-refresh", "nonCcsRefreshToken": "fresh-non-ccs-refresh",
			"idToken": "fresh-id", "expiresIn": 3599,
		})
	})
	cciMux.HandleFunc("POST /auth/v1/token-exchange", func(w http.ResponseWriter, r *http.Request) {
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

func cciTokenHandler(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]any{"accessToken": "cci-access", "refreshToken": "cci-refresh"})
}

func gateHandler(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"code":%d,"message":"Unusual activity has been detected"}`, code)
	}
}

func TestExchangeCCITokenGateFallback(t *testing.T) {
	cciMux := http.NewServeMux()
	cciMux.HandleFunc("POST /auth/v1/token", gateHandler(9035))
	cciMux.HandleFunc("POST /domain/api/v1/auth/token", cciTokenHandler)
	cciSrv := httptest.NewServer(cciMux)
	defer cciSrv.Close()

	identity := newTestIdentity(t, unreachable, cciSrv.URL)

	bundle, err := identity.exchangeCCIToken("device-abc", "testcode")
	require.NoError(t, err)
	assert.Equal(t, "cci-access", bundle.CCIAccessToken)
}

func TestExchangeCCITokenNoFallbackOnOtherErrors(t *testing.T) {
	var legacyCalls atomic.Int32

	cciMux := http.NewServeMux()
	cciMux.HandleFunc("POST /auth/v1/token", gateHandler(9013))
	cciMux.HandleFunc("POST /domain/api/v1/auth/token", func(w http.ResponseWriter, r *http.Request) {
		legacyCalls.Add(1)
		cciTokenHandler(w, r)
	})
	cciSrv := httptest.NewServer(cciMux)
	defer cciSrv.Close()

	identity := newTestIdentity(t, unreachable, cciSrv.URL)

	_, err := identity.exchangeCCIToken("device-abc", "testcode")
	require.Error(t, err)
	assert.Zero(t, legacyCalls.Load())
}

func TestExchangeCCITokenUnsignedBrandUsesLegacy(t *testing.T) {
	cciMux := http.NewServeMux()
	cciMux.HandleFunc("POST /auth/v1/token", func(w http.ResponseWriter, _ *http.Request) {
		t.Error("signed route must not be called without a fingerprint key")
	})
	cciMux.HandleFunc("POST /domain/api/v1/auth/token", func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("X-Fingerprint"))
		cciTokenHandler(w, r)
	})
	cciSrv := httptest.NewServer(cciMux)
	defer cciSrv.Close()

	identity := newTestIdentity(t, unreachable, cciSrv.URL)
	identity.config.CCI.FingerprintKey = ""

	bundle, err := identity.exchangeCCIToken("device-abc", "testcode")
	require.NoError(t, err)
	assert.Equal(t, "cci-access", bundle.CCIAccessToken)
}
