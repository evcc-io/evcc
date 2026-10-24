package ecoflow

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/logstash"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// encryptCertificate is the inverse of the server side: AES-CFB keyed by the
// SHA-256 of the token with the fixed portal IV, padded to the block size.
func encryptCertificate(t *testing.T, token string, plain []byte) string {
	t.Helper()

	n := aes.BlockSize - len(plain)%aes.BlockSize
	for range n {
		plain = append(plain, byte(n))
	}

	key := sha256.Sum256([]byte(token))
	block, err := aes.NewCipher(key[:])
	require.NoError(t, err)

	ct := make([]byte, len(plain))
	cipher.NewCFBEncrypter(block, []byte("ojsajkqjwk1w2dfg")).XORKeyStream(ct, plain)
	return base64.StdEncoding.EncodeToString(ct)
}

func TestLoginFallsBackToSecondHostAndDecryptsCertificate(t *testing.T) {
	const token = "jwt-token"

	var loginBody map[string]string

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":"1","message":"wrong region"}`))
	}))
	defer failing.Close()

	working := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&loginBody))
			_, _ = w.Write([]byte(`{"code":"0","data":{"token":"` + token + `","user":{"userId":"42"}}}`))
		case "/iot-auth/enterprise-development/user/certification":
			assert.Equal(t, "Bearer "+token, r.Header.Get("Authorization"))
			cert := encryptCertificate(t, token, []byte(`{"url":"mqtt-e.ecoflow.com","port":"8084","protocol":"wss","userName":"app-42","password":"mqtt-secret"}`))
			_, _ = w.Write([]byte(`{"code":"0","data":"` + cert + `"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer working.Close()

	defer func(orig []string) { apiBases = orig }(apiBases)
	apiBases = []string{failing.URL, working.URL}

	session, err := Login(util.NewLogger("test"), "user@example.com", "secret")
	require.NoError(t, err)

	assert.Equal(t, map[string]string{
		"email":    "user@example.com",
		"password": base64.StdEncoding.EncodeToString([]byte("secret")),
		"scene":    "IOT_APP",
		"userType": "ECOFLOW",
	}, loginBody)

	assert.Equal(t, "42", session.UserID)
	assert.Equal(t, "app-42", session.Account)
	assert.Equal(t, "mqtt-secret", session.Password)
	assert.Equal(t, working.URL, session.Base)
	assert.Equal(t, token, session.Token)
}

func TestApiHostOrder(t *testing.T) {
	assert.Equal(t, []string{"https://api-e.ecoflow.com", "https://api.ecoflow.com"}, apiBases)
}

func TestLoginFailsOnAllHosts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":"1","message":"bad credentials"}`))
	}))
	defer srv.Close()

	defer func(orig []string) { apiBases = orig }(apiBases)
	apiBases = []string{srv.URL}

	_, err := Login(util.NewLogger("test"), "user@example.com", "secret")
	assert.ErrorContains(t, err, "bad credentials")
	assert.NotContains(t, err.Error(), "secret")
}

// request.Helper traces response bodies before Login can redact the token.
func TestLoginDoesNotTraceTokenOrUserID(t *testing.T) {
	const token, userID = "untraced-token-4711", "untraced-user-0815"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			_, _ = w.Write([]byte(`{"code":"0","data":{"token":"` + token + `","user":{"userId":"` + userID + `"}}}`))
		case "/iot-auth/enterprise-development/user/certification":
			cert := encryptCertificate(t, token, []byte(`{"certificateAccount":"untraced-account","certificatePassword":"untraced-mqtt"}`))
			_, _ = w.Write([]byte(`{"code":"0","data":"` + cert + `"}`))
		}
	}))
	defer srv.Close()

	defer func(orig []string) { apiBases = orig }(apiBases)
	apiBases = []string{srv.URL}

	_, err := Login(util.NewLogger("test"), "user@example.com", "secret")
	require.NoError(t, err)

	for _, line := range logstash.All(nil, jww.LevelTrace, 0) {
		for _, secret := range []string{token, userID, "untraced-account", "untraced-mqtt"} {
			assert.NotContains(t, line, secret)
		}
	}
}
