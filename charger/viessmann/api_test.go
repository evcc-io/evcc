package viessmann

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func testAPI(t *testing.T, handler http.HandlerFunc) *API {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"})

	return NewAPI(util.NewLogger("viessmann"), srv.URL, ts)
}

func TestInstallations(t *testing.T) {
	api := testAPI(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/equipment/installations", r.URL.Path)
		require.Equal(t, "true", r.URL.Query().Get("includeGateways"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[
			{"id":3242119,"gateways":[{"serial":"7472258009383262"},{"serial":"7472258009383263"}]},
			{"id":1000001,"gateways":[{"serial":"9999999999999999"}]}
		]}`))
	})

	res, err := api.Installations()
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, 3242119, res[0].ID)
	assert.Len(t, res[0].Gateways, 2)
	assert.Equal(t, "7472258009383262", res[0].Gateways[0].Serial)
}

func TestInstallationsError(t *testing.T) {
	api := testAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := api.Installations()
	require.Error(t, err)
}
