package homeassistant

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
	return ok && a.err != nil
}

// expireRetry ends the period in which requests to an unavailable instance fail without network access
func expireRetry(uri string) {
	outageMu.Lock()
	defer outageMu.Unlock()
	outages[uri].retry = time.Time{}
}

func isPermanent(err error) bool {
	_, ok := errors.AsType[*backoff.PermanentError](err)
	return ok
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

func TestUnreachableNotRetried(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	conn := newTestConnection(srv.URL)

	var attempts int
	_, err := backoff.RetryWithData(func() (StateResponse, error) {
		attempts++
		return conn.GetState("sensor.foo")
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Millisecond), 3))

	require.Error(t, err)
	assert.Equal(t, 1, attempts)
	assert.ErrorContains(t, err, "sensor.foo", "original error message must be kept")
	assert.True(t, outage(srv.URL))
}

func TestOutage(t *testing.T) {
	var requests atomic.Int32
	var down atomic.Bool
	down.Store(true)

	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
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

	_, err := newTestConnection(srv.URL).GetState("sensor.foo")
	require.Error(t, err)
	assert.True(t, outage(srv.URL))
	assert.Equal(t, int32(1), requests.Load())

	// other connections to the same instance fail without sending a request
	_, err = newTestConnection(srv.URL).GetState("sensor.bar")
	assert.ErrorContains(t, err, srv.URL+" unavailable")
	assert.True(t, isPermanent(err))

	err = newTestConnection(srv.URL).CallSwitchService("switch.foo", true)
	assert.ErrorContains(t, err, srv.URL+" unavailable")
	assert.Equal(t, int32(1), requests.Load())

	// after the retry delay, a request reaches the instance again
	expireRetry(srv.URL)
	_, err = newTestConnection(srv.URL).GetStates()
	require.Error(t, err)
	assert.Equal(t, int32(2), requests.Load())
	assert.True(t, outage(srv.URL))

	// any response other than a gateway error proves the instance is available
	down.Store(false)
	expireRetry(srv.URL)
	_, err = newTestConnection(srv.URL).GetState("sensor.missing")
	assert.ErrorContains(t, err, "404")
	assert.False(t, outage(srv.URL))

	_, err = newTestConnection(srv.URL).GetState("sensor.foo")
	require.NoError(t, err)
	assert.Equal(t, int32(4), requests.Load())
}

// TestSingleRetry verifies that only one request reaches an unavailable instance
// after the retry delay, while the others keep failing without network access
func TestSingleRetry(t *testing.T) {
	uri := "http://single-retry"
	refused := &url.Error{Op: "Get", URL: uri, Err: &net.OpError{Op: "dial", Err: errors.New("connection refused")}}

	require.Error(t, trackAvailability(uri, func() error { return refused }))
	expireRetry(uri)

	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error)

	go func() {
		done <- trackAvailability(uri, func() error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started

	var sent bool
	err := trackAvailability(uri, func() error { sent = true; return nil })
	assert.ErrorContains(t, err, uri+" unavailable")
	assert.False(t, sent, "request must not be sent while another one is pending")

	close(release)
	require.NoError(t, <-done)
	assert.False(t, outage(uri))
}

// TestStaleResponse verifies that a response to an older request doesn't override
// the state set by a newer one, while still returning its own error
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
			assert.True(t, errors.Is(newer, tc.newer))

			close(release)
			older := <-done
			assert.True(t, errors.Is(older, tc.older), "older request keeps its own error")

			assert.Equal(t, tc.wantUnreached, outage(uri), "state follows the newer request")
		})
	}
}

// TestChangesLoggedInOrder verifies that a change queued while an earlier one is still
// being logged is logged after it, without its caller waiting for the log write
func TestChangesLoggedInOrder(t *testing.T) {
	var (
		mu     sync.Mutex
		logged []int
	)
	record := func(i int) {
		mu.Lock()
		defer mu.Unlock()
		logged = append(logged, i)
	}

	started := make(chan struct{})
	release := make(chan struct{})

	outageMu.Lock()
	changes = append(changes, func() {
		unlocked := outageMu.TryLock()
		if unlocked {
			outageMu.Unlock()
		}
		assert.True(t, unlocked, "lock must not be held while logging")

		close(started)
		if unlocked {
			<-release
		}
		record(1)
	})
	outageMu.Unlock()

	done := make(chan struct{})
	go func() {
		logChanges()
		close(done)
	}()
	<-started

	outageMu.Lock()
	changes = append(changes, func() { record(2) })
	outageMu.Unlock()

	// returns while the first change is still being logged
	logChanges()

	close(release)
	<-done

	assert.Equal(t, []int{1, 2}, logged)
}
