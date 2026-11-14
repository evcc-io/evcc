package saic

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/evcc-io/evcc/vehicle/saic/requests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestWakeupTransportError(t *testing.T) {
	log := util.NewLogger("saic")
	identity := NewIdentity(log, "", "", "http://saic.invalid/")
	identity.TokenSource = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	v := NewAPI(log, identity)

	want := errors.New("connection failed")
	v.Client.Transport = &transport.Decorator{
		Decorator: func(*http.Request) error { return want },
	}

	require.ErrorIs(t, v.Wakeup("test-vin"), want)
}

func TestWakeupHTTPResponse(t *testing.T) {
	for _, tc := range []struct {
		status    int
		mustRetry bool
		statusErr bool
	}{
		{status: http.StatusOK},
		{status: http.StatusAccepted},
		{status: http.StatusNoContent},
		{status: http.StatusServiceUnavailable, statusErr: true},
		{status: http.StatusUnauthorized, mustRetry: true},
		{status: http.StatusForbidden, mustRetry: true},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			var logins int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/oauth/token" {
					logins++
					fmt.Fprint(w, `{"code":0,"data":{"access_token":"refreshed","expires_in":3600}}`)
					return
				}
				assert.Equal(t, "/vehicle/status", r.URL.Path)
				w.WriteHeader(tc.status)
			}))
			t.Cleanup(srv.Close)

			log := util.NewLogger("saic")
			identity := NewIdentity(log, "", "", srv.URL+"/")
			identity.TokenSource = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
			v := NewAPI(log, identity)

			err := v.Wakeup("test-vin")
			switch {
			case tc.mustRetry:
				require.ErrorIs(t, err, api.ErrMustRetry)
				assert.Equal(t, 1, logins)
			case tc.statusErr:
				var statusErr *request.StatusError
				require.ErrorAs(t, err, &statusErr)
				assert.Equal(t, tc.status, statusErr.StatusCode())
			default:
				require.NoError(t, err)
			}
		})
	}
}

// TestTakeReturnsValueOnce verifies a stored value is returned exactly once,
// and every other state yields ErrMustRetry or starts a fresh request.
func TestTakeReturnsValueOnce(t *testing.T) {
	v := &API{log: util.NewLogger("saic")}

	// idle: no pending value and no poll -> caller must start a new query
	if _, _, query := v.take(); !query {
		t.Fatal("idle: expected query=true")
	}

	// background poll running -> ErrMustRetry, no new query, no value
	v.mu.Lock()
	v.state = stateRunning
	v.mu.Unlock()
	if _, err, query := v.take(); query || !errors.Is(err, api.ErrMustRetry) {
		t.Fatalf("running: got err=%v query=%v, want ErrMustRetry", err, query)
	}

	// a background value arrives
	var cs requests.ChargeStatus
	cs.RvsChargeStatus.Mileage = 4242
	v.store(cs)

	// returned exactly once with nil error
	res, err, query := v.take()
	if query || err != nil || res.RvsChargeStatus.Mileage != 4242 {
		t.Fatalf("first take: got mileage=%d err=%v query=%v, want value once", res.RvsChargeStatus.Mileage, err, query)
	}

	// consumed: the next call must start a fresh request again
	if _, _, query := v.take(); !query {
		t.Fatal("after consume: expected query=true")
	}
}
