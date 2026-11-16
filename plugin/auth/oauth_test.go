package auth

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/logstash"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestOAuth(t *testing.T) {
	var storerCalled int
	storer := func(t *oauth2.Token) any {
		storerCalled++
		return t
	}

	ts, err := NewOAuth(t.Context(), "foo", "bar", &oauth2.Config{
		ClientID: "baz",
	}, WithTokenStorerOption(storer))
	require.NoError(t, err)

	token, err := ts.Token()
	require.ErrorContains(t, err, "login required")
	require.False(t, token.Valid())
	require.Equal(t, 0, storerCalled)

	ts.updateToken(&oauth2.Token{AccessToken: "at"})
	require.Equal(t, 1, storerCalled)

	token, err = ts.Token()
	require.NoError(t, err)
	require.True(t, token.Valid())
	require.Equal(t, 1, storerCalled)
}

// TestSetOnlineNonBlocking ensures setOnline coalesces instead of blocking when
// the channel isn't drained, guarding the token-refresh deadlock.
func TestSetOnlineNonBlocking(t *testing.T) {
	o := &OAuth{onlineC: make(chan bool, 1)}
	o.setOnline(true) // fill the buffer; nobody is draining it

	done := make(chan struct{})
	go func() {
		o.setOnline(false) // would block forever on a full unbuffered/direct send
		o.setOnline(true)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("setOnline blocked while the online channel was not drained")
	}
}

func logsSince(before []string) []string {
	all := logstash.All(nil, jww.LevelTrace, 0)
	if len(before) > len(all) {
		return nil
	}
	return all[len(before):]
}

func TestRefreshExpiresIn(t *testing.T) {
	overflow := math.MaxInt64
	for _, tc := range []struct {
		name  string
		token *oauth2.Token
		d     time.Duration
		ok    bool
	}{
		{name: "nil"},
		{name: "missing", token: &oauth2.Token{}},
		{name: "float64", token: (&oauth2.Token{}).WithExtra(map[string]any{"refresh_expires_in": 1800.0}), d: 1800 * time.Second, ok: true},
		{name: "int", token: (&oauth2.Token{}).WithExtra(map[string]any{"refresh_expires_in": 1800}), d: 1800 * time.Second, ok: true},
		{name: "int64", token: (&oauth2.Token{}).WithExtra(map[string]any{"refresh_expires_in": int64(1800)}), d: 1800 * time.Second, ok: true},
		{name: "json.Number", token: (&oauth2.Token{}).WithExtra(map[string]any{"refresh_expires_in": json.Number("1800")}), d: 1800 * time.Second, ok: true},
		{name: "string", token: (&oauth2.Token{}).WithExtra(map[string]any{"refresh_expires_in": "1800"}), d: 1800 * time.Second, ok: true},
		{name: "overflow int64", token: (&oauth2.Token{}).WithExtra(map[string]any{"refresh_expires_in": overflow})},
		{name: "overflow json", token: (&oauth2.Token{}).WithExtra(map[string]any{"refresh_expires_in": json.Number("9223372036854775807")})},
		{name: "overflow string", token: (&oauth2.Token{}).WithExtra(map[string]any{"refresh_expires_in": "9223372036854775807"})},
		{name: "overflow float", token: (&oauth2.Token{}).WithExtra(map[string]any{"refresh_expires_in": 1e20})},
		{name: "NaN", token: (&oauth2.Token{}).WithExtra(map[string]any{"refresh_expires_in": math.NaN()})},
		{name: "Inf", token: (&oauth2.Token{}).WithExtra(map[string]any{"refresh_expires_in": math.Inf(1)})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, ok := refreshExpiresIn(tc.token)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.d, d)
		})
	}
}

func TestWarnShortRefresh(t *testing.T) {
	o := &OAuth{log: util.NewLogger("oauth-short"), onlineC: make(chan bool, 1)}

	before := logstash.All(nil, jww.LevelTrace, 0)
	o.warnShortRefresh((&oauth2.Token{}).WithExtra(map[string]any{"refresh_expires_in": 7776000.0}))
	assert.NotContains(t, strings.Join(logsSince(before), "\n"), "refresh token expires")

	before = logstash.All(nil, jww.LevelTrace, 0)
	o.warnShortRefresh((&oauth2.Token{}).WithExtra(map[string]any{"refresh_expires_in": 1800.0}))
	assert.Contains(t, strings.Join(logsSince(before), "\n"), "refresh token expires in 30m0s")
}

func TestTokenRefreshFailureLogged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"Token is not active"}`))
	}))
	t.Cleanup(srv.Close)

	o, err := NewOAuth(t.Context(), "foo", t.Name(), &oauth2.Config{
		ClientID:     t.Name(),
		ClientSecret: "secret",
		Endpoint:     oauth2.Endpoint{TokenURL: srv.URL},
	})
	require.NoError(t, err)

	o.mu.Lock()
	o.updateToken(&oauth2.Token{
		AccessToken:  "old",
		RefreshToken: "rt",
		Expiry:       time.Now().Add(-time.Minute),
	})
	o.mu.Unlock()
	require.True(t, settings.Exists(o.subject))

	before := logstash.All(nil, jww.LevelTrace, 0)
	_, err = o.Token()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid_grant")

	assert.Contains(t, strings.Join(logsSince(before), "\n"), "token refresh failed")
	assert.False(t, settings.Exists(o.subject))
}

func TestUpdateTokenWarnsOnShortRefresh(t *testing.T) {
	o, err := NewOAuth(t.Context(), "foo", t.Name(), &oauth2.Config{
		ClientID: t.Name() + "-short",
	})
	require.NoError(t, err)

	before := logstash.All(nil, jww.LevelTrace, 0)
	o.mu.Lock()
	o.updateToken((&oauth2.Token{AccessToken: "at", RefreshToken: "rt"}).WithExtra(map[string]any{
		"refresh_expires_in": 1800.0,
	}))
	o.mu.Unlock()
	assert.Contains(t, strings.Join(logsSince(before), "\n"), "refresh token expires in 30m0s")
}

func TestTokenRefreshFailureLoggedOnce(t *testing.T) {
	var mu sync.Mutex
	fail := true

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		_, _ = io.Copy(io.Discard, r.Body)

		mu.Lock()
		ok := !fail
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"temporarily_unavailable"}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"new","token_type":"Bearer","expires_in":3600,"refresh_token":"rt2"}`))
	}))
	t.Cleanup(srv.Close)

	o, err := NewOAuth(t.Context(), "foo", t.Name(), &oauth2.Config{
		ClientID:     t.Name(),
		ClientSecret: "secret",
		Endpoint:     oauth2.Endpoint{TokenURL: srv.URL},
	})
	require.NoError(t, err)

	expired := &oauth2.Token{
		AccessToken:  "old",
		RefreshToken: "rt",
		Expiry:       time.Now().Add(-time.Minute),
	}
	o.mu.Lock()
	o.updateToken(expired)
	o.mu.Unlock()

	before := logstash.All(nil, jww.LevelTrace, 0)
	_, err = o.Token()
	require.Error(t, err)
	_, err = o.Token()
	require.Error(t, err)
	logs := strings.Join(logsSince(before), "\n")
	assert.Equal(t, 1, strings.Count(logs, "token refresh failed"))

	mu.Lock()
	fail = false
	mu.Unlock()
	tok, err := o.Token()
	require.NoError(t, err)
	require.Equal(t, "new", tok.AccessToken)

	mu.Lock()
	fail = true
	mu.Unlock()
	o.mu.Lock()
	o.token.Expiry = time.Now().Add(-time.Minute)
	o.mu.Unlock()

	before = logstash.All(nil, jww.LevelTrace, 0)
	_, err = o.Token()
	require.Error(t, err)
	assert.Equal(t, 1, strings.Count(strings.Join(logsSince(before), "\n"), "token refresh failed"))
}
