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
			{"id":3242119,"description":"Home","address":{"street":"Main St","houseNumber":"1","zip":"10115","city":"Berlin","country":"de"},"gateways":[{"serial":"7472258009383262","version":"1.4.0.0","gatewayType":"VitoconnectOPTO2"},{"serial":"7472258009383263","gatewayType":"TCU301"}]},
			{"id":1000001,"description":null,"address":{"street":null,"city":null},"gateways":[{"serial":"9999999999999999"}]}
		]}`))
	})

	res, err := api.Installations()
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, 3242119, res[0].ID)
	assert.Equal(t, "Home", res[0].Description)
	assert.Equal(t, Address{Street: "Main St", HouseNumber: "1", City: "Berlin"}, res[0].Address)
	assert.Len(t, res[0].Gateways, 2)
	assert.Equal(t, Gateway{Serial: "7472258009383262", GatewayType: "VitoconnectOPTO2"}, res[0].Gateways[0])
	assert.Equal(t, Installation{ID: 1000001, Gateways: []Gateway{{Serial: "9999999999999999"}}}, res[1])
}

func TestInstallationsError(t *testing.T) {
	api := testAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := api.Installations()
	require.Error(t, err)
}

func TestFeatures(t *testing.T) {
	api := testAPI(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/features/installations/3242119/gateways/7472258009383262/devices/0/features", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[
			{"feature":"heating.dhw.oneTimeCharge","properties":{"active":{"type":"boolean","value":false}}},
			{"feature":"heating.power.consumption.current","properties":{"value":{"type":"number","value":1.345}}}
		]}`))
	})

	res, err := api.Features("3242119", "7472258009383262", "0")
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, "heating.dhw.oneTimeCharge", res[0].Feature)
	assert.Equal(t, "heating.power.consumption.current", res[1].Feature)
}

func TestFeaturesError(t *testing.T) {
	api := testAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := api.Features("3242119", "7472258009383262", "0")
	require.Error(t, err)
}
