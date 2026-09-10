package hello

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// TestRetryTokenInvalid ensures a 1402 response triggers a fresh login and a retry
// instead of failing the request.
func TestRetryTokenInvalid(t *testing.T) {
	var requests int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Write([]byte(`{"code":"1402","message":"The access token is invalid"}`))
			return
		}
		w.Write([]byte(`{"code":"1000","data":{"list":[{"vin":"VIN"}]}}`))
	}))
	defer srv.Close()

	log := util.NewLogger("test")

	var logins int
	identity := &Identity{
		Helper: request.NewHelper(log),
		log:    log,
		userID: "user",
	}
	identity.refresher = func(*oauth2.Token) (*oauth2.Token, error) {
		logins++
		return &oauth2.Token{AccessToken: "fresh", Expiry: time.Now().Add(time.Hour)}, nil
	}
	identity.ts = oauth.RefreshTokenSource(log, &oauth2.Token{AccessToken: "stale", Expiry: time.Now().Add(time.Hour)}, identity.refresher)

	api := NewAPI(log, identity)
	api.baseURI = srv.URL

	vehicles, err := api.Vehicles()
	require.NoError(t, err)
	require.Len(t, vehicles, 1)
	require.Equal(t, 2, requests)
	require.Equal(t, 1, logins)
}
