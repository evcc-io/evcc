package plugin

import (
	"errors"
	"math/rand/v2"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestWatchdogSetterConcurrency(t *testing.T) {
	p := &watchdogPlugin{
		log:     util.NewLogger("foo"),
		timeout: 10 * time.Nanosecond,
	}

	var u atomic.Uint32

	set := setter(p, func(i int) error {
		if !u.CompareAndSwap(0, 1) {
			return errors.New("race")
		}

		time.Sleep(time.Duration(rand.Int32N(int32(p.timeout))))

		if !u.CompareAndSwap(1, 0) {
			return errors.New("race")
		}

		return nil
	}, nil)

	var eg errgroup.Group

	for range 100 {
		eg.Go(func() error {
			return set(1)
		})
	}

	require.NoError(t, eg.Wait())
}

func TestWatchdogDelayTransition(t *testing.T) {
	// Test: Mode 1 → 3 → 2 with delay
	// 1 → 3: delayed (target 3 is non-reset)
	// 3 → 2: delayed (target 2 is non-reset)
	// Expected: [1, <delay>, 3, <delay>, 2]

	timeout := 100 * time.Millisecond
	p := &watchdogPlugin{
		log:        util.NewLogger("test"),
		timeout:    timeout,
		transition: true,
	}

	var calls []int
	var timestamps []time.Time

	set := setter(p, func(i int) error {
		calls = append(calls, i)
		timestamps = append(timestamps, time.Now())
		return nil
	}, []int{1}) // 1 is reset mode

	// Mode 1 (reset) → should write immediately
	require.NoError(t, set(1))
	require.Equal(t, []int{1}, calls)

	// Mode 3 (target is non-reset) → should be delayed
	require.NoError(t, set(3))
	require.Equal(t, []int{1}, calls, "Mode 3 should not be written yet")

	// Wait for delay + small buffer
	expectedDelay := timeout + 5*time.Second
	time.Sleep(expectedDelay + 20*time.Millisecond)

	// Now mode 3 should be written
	require.Equal(t, []int{1, 3}, calls)

	// Mode 2 (non-reset to non-reset) → should delay
	require.NoError(t, set(2))
	require.Equal(t, []int{1, 3}, calls, "Mode 2 should not be written yet")

	// Wait for delay + small buffer
	time.Sleep(expectedDelay + 20*time.Millisecond)

	// Now mode 2 should be written (exactly once)
	require.Equal(t, []int{1, 3, 2}, calls)

	// Verify delay was approximately correct for mode 2
	if len(timestamps) >= 3 {
		actualDelay := timestamps[2].Sub(timestamps[1])
		require.Greater(t, actualDelay, expectedDelay-10*time.Millisecond)
		require.Less(t, actualDelay, expectedDelay+100*time.Millisecond)
	}
}

func TestWatchdogCancelPendingTransition(t *testing.T) {
	// Test: Mode 3 → 2 started, then set Mode 1 during delay
	// Expected: Transition cancelled, Mode 1 set immediately

	timeout := 200 * time.Millisecond
	p := &watchdogPlugin{
		log:        util.NewLogger("test"),
		timeout:    timeout,
		transition: true,
	}

	var calls []int
	set := setter(p, func(i int) error {
		calls = append(calls, i)
		return nil
	}, []int{1}) // 1 is reset mode

	// Mode 3 (non-reset)
	require.NoError(t, set(3))
	require.Equal(t, []int{3}, calls)

	// Mode 2 (delayed transition)
	require.NoError(t, set(2))
	require.Equal(t, []int{3}, calls, "Mode 2 should not be written yet")

	// Wait a bit but not the full delay
	time.Sleep(50 * time.Millisecond)

	// Mode 1 (reset) → should cancel pending transition and write immediately
	require.NoError(t, set(1))
	require.Equal(t, []int{3, 1}, calls, "Mode 1 should be written, Mode 2 should be cancelled")

	// Wait for what would have been the original delay
	time.Sleep(timeout + 5*time.Second + 100*time.Millisecond)

	// Mode 2 should still not have been written
	require.Equal(t, []int{3, 1}, calls, "Mode 2 should remain cancelled")
}

func TestWatchdogDelayBackwardCompatibility(t *testing.T) {
	// Test: transition=false behaves like old implementation
	// Expected: All transitions immediate

	p := &watchdogPlugin{
		log:        util.NewLogger("test"),
		timeout:    100 * time.Millisecond,
		transition: false, // explicitly false
	}

	var calls []int
	set := setter(p, func(i int) error {
		calls = append(calls, i)
		return nil
	}, []int{1}) // 1 is reset mode

	// All transitions should be immediate
	require.NoError(t, set(1))
	require.Equal(t, []int{1}, calls)

	require.NoError(t, set(3))
	require.Equal(t, []int{1, 3}, calls)

	require.NoError(t, set(2))
	require.Equal(t, []int{1, 3, 2}, calls, "Mode 2 should be written immediately (no delay)")

	require.NoError(t, set(4))
	require.Equal(t, []int{1, 3, 2, 4}, calls, "Mode 4 should be written immediately (no delay)")
}


