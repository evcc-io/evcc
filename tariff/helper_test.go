package tariff

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
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

func (r *runner) run(done chan error) {
	if r.res == nil {
		close(done)
	} else {
		done <- r.res
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

// leakRunner mirrors the real tariff run loop: its first update always fails
// (e.g. HTTP 429 at startup) and it reports the error via reportError. With the
// fix in place reportError returns true and run returns; without it the loop
// would block on the hourly tick and keep polling forever.
type leakRunner struct {
	exited chan struct{}
}

func (r *leakRunner) run(done chan error) {
	defer close(r.exited)

	var once sync.Once
	for tick := time.Tick(time.Hour); ; <-tick {
		if reportError(&once, done, errors.New("initial failure (e.g. HTTP 429)")) {
			return
		}
		// without the fix the goroutine would block on <-tick here and keep
		// hitting the API on every tick - exactly the leak we guard against
	}
}

// TestRunOrErrorStopsGoroutineOnStartupFailure asserts that when the first
// update fails, runOrError both propagates the error and stops the background
// goroutine instead of leaving it to poll the API indefinitely.
func TestRunOrErrorStopsGoroutineOnStartupFailure(t *testing.T) {
	r := &leakRunner{exited: make(chan struct{})}

	res, err := runOrError(r)
	require.Error(t, err, "startup failure must be propagated")
	require.Nil(t, res, "tariff must not be returned when startup fails")

	select {
	case <-r.exited:
		// goroutine returned - no leak
	case <-time.After(time.Second):
		t.Fatal("run goroutine still alive after startup failure (goroutine leak)")
	}
}
