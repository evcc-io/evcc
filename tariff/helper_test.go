package tariff

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeRatesAfter(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	rate := func(start int, val float64) api.Rate {
		return api.Rate{
			Start: clock.Now().Add(time.Duration(start) * time.Hour),
			End:   clock.Now().Add(time.Duration(start+1) * time.Hour),
			Value: val,
		}
	}

	old := api.Rates{rate(1, 1), rate(2, 2)}
	new := api.Rates{rate(2, 2), rate(3, 3)}
	combined := api.Rates{rate(1, 1), rate(2, 2), rate(3, 3)}

	data := util.NewMonitor[api.Rates](time.Hour).WithClock(clock)

	data.Set(old)
	res, err := data.Get()
	require.NoError(t, err)
	assert.Equal(t, old, res)

	for i, tc := range []struct {
		new, expected api.Rates
		ts            time.Time
	}{
		{new, combined, clock.Now()},
		{new, combined, clock.Now().Add(SlotDuration)},
		{nil, combined, clock.Now()},
		{nil, combined, clock.Now().Add(SlotDuration)},
		{new, combined, clock.Now().Add(time.Hour)},
		{new, combined, clock.Now().Add(time.Hour + SlotDuration)},
		{new, combined, now.With(clock.Now().Add(time.Hour + 30*time.Minute)).BeginningOfHour()},
		{new, combined, now.With(clock.Now().Add(time.Hour + 30*time.Minute)).BeginningOfHour().Add(SlotDuration)},
		{new, new, clock.Now().Add(2 * time.Hour)},
		{new, new, clock.Now().Add(2*time.Hour + SlotDuration)},
	} {
		t.Logf("%d. %+v", i+1, tc)

		mergeRatesAfter(data, tc.new, tc.ts)

		res, err := data.Get()
		require.NoError(t, err)
		assert.Equal(t, tc.expected, res)
	}
}

type runner struct {
	res error
}

func (r *runner) run(done chan error, _ <-chan struct{}) {
	if r.res == nil {
		close(done)
	} else {
		done <- r.res
	}
}

// persistingRunner mimics the error branch of GrünStromIndex.run:
// it sends the initial error to done (so runOrError propagates it and discards
// the tariff), then keeps running — just like the real loop does via `continue`.
// The stop channel gives the test a way to clean up; without it the goroutine
// would truly leak, which is exactly the problem we are demonstrating.
type persistingRunner struct {
	stop    chan struct{}
	running chan struct{} // closed by goroutine after the initial failure
}

func (r *persistingRunner) run(done chan error, stop <-chan struct{}) {
	done <- errors.New("initial failure (e.g. HTTP 429)")
	// Simulate the real tariff behaviour: after sending the error, the goroutine
	// calls `continue` and then blocks on `<-tick` (up to one hour) before doing
	// anything else. During that waiting window, runOrError's close(stop) call has
	// plenty of time to arrive. We replicate this with a short timer: if stop is
	// closed quickly (fix is in place) we exit cleanly; if stop is never closed
	// (bug is still present) the timer fires and we signal the leak.
	select {
	case <-stop:
		return // correctly stopped — no leak
	case <-time.After(100 * time.Millisecond):
		close(r.running) // stop never closed — goroutine leak
		<-r.stop
	}
}

func TestRunOrQError(t *testing.T) {
	{
		res, err := runOrError(&runner{nil})
		require.NoError(t, err)
		require.NotNil(t, res)
	}
	{
		res, err := runOrError(&runner{errors.New("foo")})
		require.Error(t, err)
		require.Nil(t, res)
	}
}

// TestRunOrError_DoesNotLeakGoroutineOnInitialFailure asserts that when the
// first API call inside run() fails (e.g. HTTP 429 at startup), runOrError must
// stop the background goroutine before returning the error.
//
// Without that guarantee the goroutine outlives the tariff: it keeps hitting
// the API on every hourly tick even though evcc has already discarded the
// tariff object and will never read the results. For GSI this is particularly
// harmful: the orphaned goroutine burns rate-limit quota, so an evcc restart
// cannot heal the situation — the provider keeps seeing traffic and keeps
// returning 429.
func TestRunOrError_DoesNotLeakGoroutineOnInitialFailure(t *testing.T) {
	r := &persistingRunner{
		stop:    make(chan struct{}),
		running: make(chan struct{}),
	}
	// Ensure the goroutine is stopped at the end of the test regardless.
	t.Cleanup(func() { close(r.stop) })

	result, err := runOrError(r)
	require.Error(t, err, "runOrError must propagate the initial failure")
	require.Nil(t, result, "the tariff must not be registered when initialisation fails")

	// If runOrError properly stopped the goroutine, r.running will never close
	// and we will hit the timeout branch below (pass). If the goroutine is still
	// alive it closes r.running immediately, which triggers the failure branch.
	select {
	case <-r.running:
		t.Error("goroutine is still running after runOrError returned an error: " +
			"the background goroutine must be cancelled when the tariff fails to initialise, " +
			"otherwise it keeps hitting the API indefinitely (goroutine leak)")
	case <-time.After(50 * time.Millisecond):
		// goroutine has stopped — no leak, desired behaviour
	}
}

// statusErrorFor creates a *request.StatusError for the given HTTP status code,
// matching what request.Helper.GetJSON returns for non-2xx responses.
func statusErrorFor(code int) error {
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp := &http.Response{
		StatusCode: code,
		Request:    req,
	}
	return request.NewStatusError(resp)
}

// TestBackoffPermanentError_429IsTransient confirms that HTTP 429 (Too Many Requests)
// is NOT treated as a permanent error by backoffPermanentError.
//
// A 429 is a transient rate-limit signal: the caller should back off and retry.
// Marking it permanent causes backoff.Retry to give up immediately, and when this
// happens on the very first API call during startup the tariff is never registered
// (runOrError propagates the error and discards the tariff object). See issue #26654.
func TestBackoffPermanentError_429IsTransient(t *testing.T) {
	err := backoffPermanentError(statusErrorFor(http.StatusTooManyRequests))

	var pe *backoff.PermanentError
	assert.False(t, errors.As(err, &pe),
		"HTTP 429 must not be permanent: it is transient and backoff.Retry should keep retrying")
}

// TestBackoffPermanentError_4xxIsPermanent confirms that other 4xx errors (e.g. 400
// Bad Request) remain permanent — they will not succeed on retry.
func TestBackoffPermanentError_4xxIsPermanent(t *testing.T) {
	for _, code := range []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
	} {
		err := backoffPermanentError(statusErrorFor(code))

		var pe *backoff.PermanentError
		assert.True(t, errors.As(err, &pe),
			"HTTP %d should be permanent (won't succeed on retry)", code)
	}
}
