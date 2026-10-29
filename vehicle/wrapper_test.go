package vehicle

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	corevehicle "github.com/evcc-io/evcc/core/vehicle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// retryVehicle is a vehicle with charge state
type retryVehicle struct {
	*embed
}

func (v *retryVehicle) Soc() (float64, error) {
	return 42, nil
}

func (v *retryVehicle) Status() (api.ChargeStatus, error) {
	return api.StatusB, nil
}

var retryable struct {
	mu    sync.Mutex
	err   error
	calls int
}

func init() {
	registry.AddCtx("test-retry", func(context.Context, map[string]any) (api.Vehicle, error) {
		retryable.mu.Lock()
		defer retryable.mu.Unlock()

		retryable.calls++
		if retryable.err != nil {
			return nil, retryable.err
		}
		return &retryVehicle{&embed{Features_: []api.Feature{api.Streaming}}}, nil
	})
}

func setRetryErr(err error) {
	retryable.mu.Lock()
	defer retryable.mu.Unlock()
	retryable.err = err
}

func retryCalls() int {
	retryable.mu.Lock()
	defer retryable.mu.Unlock()
	return retryable.calls
}

func TestWrapperRetry(t *testing.T) {
	var published atomic.Int32
	corevehicle.Publish = func() { published.Add(1) }
	t.Cleanup(func() { corevehicle.Publish = nil })

	setRetryErr(errors.New("unavailable"))

	_, err := NewFromConfig(context.TODO(), "test-retry", nil)
	require.Error(t, err)

	res := NewWrapper(context.TODO(), "my car", "test-retry", map[string]any{"capacity": 50}, err)
	w := res.(*Wrapper)

	// creation has just failed: no immediate retry
	calls := retryCalls()

	_, err = res.Soc()
	require.ErrorContains(t, err, "vehicle not available")
	assert.Equal(t, []api.Feature{api.Offline, api.Retryable}, res.Features())
	assert.Equal(t, "My Car", res.GetTitle())
	assert.Equal(t, 50.0, res.Capacity())
	assert.False(t, api.HasCap[api.ChargeState](res))
	assert.Equal(t, calls, retryCalls())

	// retry interval elapsed but vehicle still unavailable: one background attempt
	w.mu.Lock()
	w.retriedAt = time.Time{}
	w.mu.Unlock()

	_ = res.Features()
	_ = res.Features()
	require.Eventually(t, func() bool { return retryCalls() == calls+1 }, time.Second, 10*time.Millisecond)

	_, err = res.Soc()
	require.ErrorContains(t, err, "vehicle not available")
	assert.Contains(t, res.Features(), api.Offline)
	assert.Equal(t, calls+1, retryCalls())

	// vehicle becomes available: next attempt creates and publishes it
	setRetryErr(nil)

	w.mu.Lock()
	w.retriedAt = time.Time{}
	w.mu.Unlock()

	_ = res.Features()
	require.Eventually(t, func() bool { return published.Load() == 1 }, time.Second, 10*time.Millisecond)

	soc, err := res.Soc()
	require.NoError(t, err)
	assert.Equal(t, 42.0, soc)
	assert.Equal(t, []api.Feature{api.Streaming}, res.Features())
	assert.Equal(t, "My Car", res.GetTitle())
	assert.True(t, api.HasCap[api.ChargeState](res))
	assert.Equal(t, calls+2, retryCalls())
}
