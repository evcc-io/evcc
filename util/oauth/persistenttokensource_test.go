package oauth

import (
	"errors"
	"testing"
	"time"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
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
		return nil, errors.New(`oauth2: "invalid_grant" refresh token is invalid`)
	})
	require.ErrorIs(t, err, ErrTokenExpired)

	_, err = settings.String(key)
	require.ErrorIs(t, err, settings.ErrNotFound)
}
