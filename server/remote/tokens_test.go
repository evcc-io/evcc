package remote

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenLifecycle(t *testing.T) {
	require.NoError(t, settings.SetJson(keys.RemoteClients, []persistedClient{}))
	r := &Remote{}

	_, pw, err := r.CreateClient("app", 0)
	require.NoError(t, err)

	_, ok := r.ValidateToken("unknown")
	assert.False(t, ok)

	token, expires, err := r.IssueToken("app")
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(tokenLifetime), expires, time.Minute)

	user, ok := r.ValidateToken(token)
	assert.True(t, ok)
	assert.Equal(t, "app", user)

	// token expiry capped by client expiry
	_, _, err = r.CreateClient("short", time.Hour)
	require.NoError(t, err)
	_, expires, err = r.IssueToken("short")
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(time.Hour), expires, time.Minute)

	// deleting the client revokes its tokens
	require.NoError(t, r.DeleteClient("app"))
	_, ok = r.ValidateToken(token)
	assert.False(t, ok)
	assert.False(t, r.Authenticate("app", pw))
}

func TestTokenMiddleware(t *testing.T) {
	require.NoError(t, settings.SetJson(keys.RemoteClients, []persistedClient{}))
	r := &Remote{}
	_, pw, err := r.CreateClient("app", 0)
	require.NoError(t, err)

	tun := NewTunnel("", "", nil, r, nil, util.NewLogger("test"), nil)
	handler := tun.basicAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong"))
	}))

	do := func(req *http.Request) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	// token endpoint requires basic auth
	req := httptest.NewRequest(http.MethodPost, tokenPath, nil)
	assert.Equal(t, http.StatusUnauthorized, do(req).Code)

	req = httptest.NewRequest(http.MethodPost, tokenPath, nil)
	req.SetBasicAuth("app", pw)
	rec := do(req)
	require.Equal(t, http.StatusOK, rec.Code)

	var res struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	assert.Equal(t, "Bearer", res.TokenType)
	assert.NotEmpty(t, res.AccessToken)
	assert.Greater(t, res.ExpiresIn, 0)

	// bearer token grants access without basic auth
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+res.AccessToken)
	rec = do(req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "pong", rec.Body.String())

	// invalid bearer token is rejected
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer nope")
	assert.Equal(t, http.StatusUnauthorized, do(req).Code)
}
