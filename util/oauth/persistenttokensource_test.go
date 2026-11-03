package oauth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// storeToken persists token under a key unique to the test. Settings are in-memory
// without a database, so blanking the key on cleanup is enough to isolate tests.
func storeToken(t *testing.T, token *oauth2.Token) string {
	t.Helper()

	key := "test." + t.Name()
	t.Cleanup(func() { settings.SetString(key, "") })

	require.NoError(t, settings.SetJson(key, token))

	return key
}

func TestPersistedTokenPrecedence(t *testing.T) {
	key := storeToken(t, &oauth2.Token{
		AccessToken: "persisted",
		Expiry:      time.Now().Add(time.Hour),
	})

	ts, err := PersistentTokenSource(util.NewLogger("test"), key, &oauth2.Token{
		AccessToken: "configured",
		Expiry:      time.Now().Add(time.Hour),
	}, func(*oauth2.Token) (*oauth2.Token, error) {
		return nil, errors.New("unexpected refresh")
	})
	require.NoError(t, err)

	token, err := ts.Token()
	require.NoError(t, err)
	require.Equal(t, "persisted", token.AccessToken)
}

func TestRefreshedTokenPersisted(t *testing.T) {
	key := storeToken(t, &oauth2.Token{
		AccessToken:  "old",
		RefreshToken: "refresh",
		Expiry:       time.Now().Add(-time.Hour),
	})

	_, err := PersistentTokenSource(util.NewLogger("test"), key, nil, func(token *oauth2.Token) (*oauth2.Token, error) {
		require.Equal(t, "refresh", token.RefreshToken)
		return &oauth2.Token{AccessToken: "new", Expiry: time.Now().Add(time.Hour)}, nil
	})
	require.NoError(t, err)

	var stored oauth2.Token
	require.NoError(t, settings.Json(key, &stored))
	require.Equal(t, "new", stored.AccessToken)
	require.Equal(t, "refresh", stored.RefreshToken, "refresh token not carried over")
}

func TestRejectedTokenDropped(t *testing.T) {
	key := storeToken(t, &oauth2.Token{
		AccessToken:  "old",
		RefreshToken: "refresh",
		Expiry:       time.Now().Add(-time.Hour),
	})

	_, err := PersistentTokenSource(util.NewLogger("test"), key, nil, func(*oauth2.Token) (*oauth2.Token, error) {
		return nil, &oauth2.RetrieveError{ErrorCode: "invalid_grant"}
	})
	require.ErrorIs(t, err, ErrTokenExpired)

	_, err = settings.String(key)
	require.ErrorIs(t, err, settings.ErrNotFound)
}

// TestRejectedStatusErrorTokenDropped covers apis refreshing via request.Helper, which
// does not return a typed oauth2 error: the error code only exists in the response body
func TestRejectedStatusErrorTokenDropped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()

	key := storeToken(t, &oauth2.Token{
		AccessToken:  "old",
		RefreshToken: "refresh",
		Expiry:       time.Now().Add(-time.Hour),
	})

	_, err := PersistentTokenSource(util.NewLogger("test"), key, nil, func(*oauth2.Token) (*oauth2.Token, error) {
		var res oauth2.Token
		return nil, request.NewHelper(util.NewLogger("test")).GetJSON(srv.URL, &res)
	})
	require.ErrorIs(t, err, ErrTokenExpired)

	_, err = settings.String(key)
	require.ErrorIs(t, err, settings.ErrNotFound)
}

// TestRejectedConfigTokenKeepsPersisted ensures a rejected config token does not discard
// a persisted token that failed for a transient reason
func TestRejectedConfigTokenKeepsPersisted(t *testing.T) {
	key := storeToken(t, &oauth2.Token{
		AccessToken:  "old",
		RefreshToken: "persisted",
		Expiry:       time.Now().Add(-time.Hour),
	})

	_, err := PersistentTokenSource(util.NewLogger("test"), key, &oauth2.Token{
		RefreshToken: "configured",
	}, func(token *oauth2.Token) (*oauth2.Token, error) {
		if token.RefreshToken == "persisted" {
			return nil, errors.New("api not reachable")
		}
		return nil, &oauth2.RetrieveError{ErrorCode: "invalid_grant"}
	})
	require.ErrorIs(t, err, ErrTokenExpired)

	var stored oauth2.Token
	require.NoError(t, settings.Json(key, &stored))
	require.Equal(t, "persisted", stored.RefreshToken)
}
