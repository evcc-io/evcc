package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
	d, ok := refreshExpiresIn(nil)
	assert.False(t, ok)
	assert.Zero(t, d)

	d, ok = refreshExpiresIn(&oauth2.Token{})
	assert.False(t, ok)

	for _, extra := range []any{1800.0, 1800, int64(1800), json.Number("1800"), "1800"} {
		tok := (&oauth2.Token{}).WithExtra(map[string]any{"refresh_expires_in": extra})
		d, ok = refreshExpiresIn(tok)
		assert.True(t, ok, "%T", extra)
		assert.Equal(t, 1800*time.Second, d, "%T", extra)
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
