package homeassistant

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func outage(uri string) bool {
	outageMu.Lock()
	defer outageMu.Unlock()
	a, ok := outages[uri]
	return ok && a.down
}

func TestIsUnreachable(t *testing.T) {
	statusError := func(code int) error {
		req := httptest.NewRequest(http.MethodGet, "http://ha/api/states/sensor.foo", nil)
		return request.NewStatusError(&http.Response{StatusCode: code, Request: req})
	}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"connection refused", &url.Error{Op: "Get", URL: "http://ha", Err: &net.OpError{Op: "dial", Err: errors.New("connection refused")}}, true},
		{"timeout", &url.Error{Op: "Get", URL: "http://ha", Err: &net.DNSError{IsTimeout: true}}, true},
		{"bad gateway", statusError(http.StatusBadGateway), true},
		{"service unavailable", statusError(http.StatusServiceUnavailable), true},
		{"not found", statusError(http.StatusNotFound), false},
		{"login required", &url.Error{Op: "Get", URL: "http://ha", Err: api.LoginRequiredError("homeassistant")}, false},
		{"invalid state", api.ErrNotAvailable, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isUnreachable(tc.err))
		})
	}
}

func TestGetStateUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	conn := newTestConnection(srv.URL)

	_, err := conn.GetState("sensor.foo")
	require.ErrorIs(t, err, api.ErrUnreachable)
	assert.ErrorContains(t, err, "sensor.foo", "original error message must be kept")
	assert.True(t, outage(srv.URL))

	// other connections to the same instance share the outage
	err = newTestConnection(srv.URL).CallSwitchService("switch.foo", true)
	require.ErrorIs(t, err, api.ErrUnreachable)
	assert.True(t, outage(srv.URL))
}

func TestAvailabilityRecovery(t *testing.T) {
	var down atomic.Bool

	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case down.Load():
			w.WriteHeader(http.StatusBadGateway)
		case r.URL.Path == "/api/states/sensor.missing":
			w.WriteHeader(http.StatusNotFound)
		default:
			fmt.Fprint(w, `{"entity_id":"sensor.foo","state":"1"}`)
		}
	}))
	srv.Start()

	conn := newTestConnection(srv.URL)

	_, err := conn.GetState("sensor.foo")
	require.NoError(t, err)
	assert.False(t, outage(srv.URL))

	down.Store(true)
	_, err = conn.GetState("sensor.foo")
	require.ErrorIs(t, err, api.ErrUnreachable)
	assert.True(t, outage(srv.URL))

	// status errors are permanent, the marker must survive backoff unwrapping them
	_, err = backoff.RetryWithData(func() (StateResponse, error) {
		return conn.GetState("sensor.foo")
	}, backoff.NewExponentialBackOff())
	require.ErrorIs(t, err, api.ErrUnreachable)

	// any response other than a gateway error proves the instance is available
	down.Store(false)
	_, err = conn.GetState("sensor.missing")
	require.Error(t, err)
	assert.NotErrorIs(t, err, api.ErrUnreachable)
	assert.False(t, outage(srv.URL))
}

// TestStaleResponse verifies that a response to an older request doesn't override
// the state set by a newer one, while still returning its own marked error
func TestStaleResponse(t *testing.T) {
	refused := &url.Error{Op: "Get", URL: "http://ha", Err: &net.OpError{Op: "dial", Err: errors.New("connection refused")}}

	tests := []struct {
		name          string
		older, newer  error
		wantUnreached bool
	}{
		{"older failure after newer success", refused, nil, false},
		{"older success after newer failure", nil, refused, true},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uri := fmt.Sprintf("http://stale%d", i)

			started := make(chan struct{})
			release := make(chan struct{})
			done := make(chan error)

			go func() {
				done <- trackAvailability(uri, func() error {
					close(started)
					<-release
					return tc.older
				})
			}()
			<-started

			newer := trackAvailability(uri, func() error { return tc.newer })
			assert.Equal(t, tc.newer != nil, errors.Is(newer, api.ErrUnreachable))

			close(release)
			older := <-done
			assert.Equal(t, tc.older != nil, errors.Is(older, api.ErrUnreachable), "older request keeps its own error")

			assert.Equal(t, tc.wantUnreached, outage(uri), "state follows the newer request")
		})
	}
}

func TestGetStatesTracked(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	conn := newTestConnection(srv.URL)

	_, err := conn.GetStates()
	require.ErrorIs(t, err, api.ErrUnreachable)

	_, err = conn.GetServices()
	require.ErrorIs(t, err, api.ErrUnreachable)

	assert.True(t, outage(srv.URL))
}
