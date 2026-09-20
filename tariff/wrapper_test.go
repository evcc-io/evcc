package tariff

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// retryTariff fails instantiation on demand
type retryTariff struct {
	mu  sync.Mutex
	err error
}

func (t *retryTariff) setErr(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.err = err
}

func (t *retryTariff) Rates() (api.Rates, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.err != nil {
		return nil, t.err
	}
	now := time.Now().Truncate(SlotDuration)
	return api.Rates{{Start: now, End: now.Add(SlotDuration), Value: 1}}, nil
}

func (t *retryTariff) Type() api.TariffType {
	return api.TariffTypePriceForecast
}

var retryable = new(retryTariff)

func init() {
	registry.AddCtx("test-retry", func(context.Context, map[string]any) (api.Tariff, error) {
		if _, err := retryable.Rates(); err != nil {
			return nil, err
		}
		return retryable, nil
	})
}

func TestWrapperRetry(t *testing.T) {
	retryable.setErr(errors.New("unavailable"))

	_, err := NewFromConfig(context.TODO(), "test-retry", nil)
	require.Error(t, err)

	res := NewWrapper(context.TODO(), "test-retry", nil, err)
	w := res.(*Wrapper)

	_, err = res.Rates()
	require.ErrorContains(t, err, "tariff not available")
	assert.Equal(t, api.TariffType(0), res.Type())

	// tariff becomes available but retry interval not elapsed: still unavailable
	retryable.setErr(nil)
	_, err = res.Rates()
	require.Error(t, err)

	// retry interval elapsed: next call creates the tariff in the background
	w.mu.Lock()
	w.retryAt = time.Time{}
	w.mu.Unlock()

	require.Eventually(t, func() bool {
		rr, err := res.Rates()
		return err == nil && len(rr) == 1
	}, time.Second, 10*time.Millisecond)
	assert.Equal(t, api.TariffTypePriceForecast, res.Type())

	// tariff fails at runtime: error passed through, no re-creation
	retryable.setErr(api.ErrOutdated)
	_, err = res.Rates()
	require.ErrorIs(t, err, api.ErrOutdated)
	assert.Same(t, retryable, w.tariff)
}
