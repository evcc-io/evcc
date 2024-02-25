package tesla

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/evcc-io/tesla-proxy-client"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestCommandResponse(t *testing.T) {
	sponsor.Subject = "any"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.URL.Path)

		switch r.URL.Path {
		case "/api/1/vehicles/abc":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"response": {"vin": "abc"}}`))

		case "/api/1/vehicles/abc/command/charge_start":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"response": null, "error": "vehicle unavailable: vehicle is offline or asleep"}`))
		}
	}))
	defer srv.Close()

	ts := oauth2.StaticTokenSource(new(oauth2.Token))
	client, err := tesla.NewClient(context.Background(), tesla.WithTokenSource(ts))
	require.NoError(t, err)
	client.SetBaseUrl(srv.URL)

	v, err := client.Vehicle("abc")
	require.NoError(t, err)

	require.ErrorIs(t, NewController(v).StartCharge(), api.ErrAsleep)
}
