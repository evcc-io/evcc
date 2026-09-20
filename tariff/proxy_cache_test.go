package tariff

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/db/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// flakyTariff fails instantiation and rate retrieval on demand
type flakyTariff struct {
	mu    sync.Mutex
	err   error
	rates api.Rates
}

func (t *flakyTariff) setErr(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.err = err
}

func (t *flakyTariff) Rates() (api.Rates, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.rates, t.err
}

func (t *flakyTariff) Type() api.TariffType {
	return api.TariffTypePriceForecast
}

var flaky = new(flakyTariff)

func init() {
	registry.AddCtx("test-cached", func(context.Context, map[string]any) (api.Tariff, error) {
		if _, err := flaky.Rates(); err != nil {
			return nil, err
		}
		return flaky, nil
	})
}

func TestCachedFallbackAndRetry(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	// utc for comparing cached rates after json round trip
	now := time.Now().UTC().Truncate(SlotDuration)
	live := makeRates(now, SlotDuration, 4, 10)
	stale := makeRates(now.Add(-time.Hour), SlotDuration, 4, 0)

	tariff := flaky
	tariff.mu.Lock()
	tariff.rates = live
	tariff.err = errors.New("unavailable")
	tariff.mu.Unlock()

	other := map[string]any{"interval": 50 * time.Millisecond}
	key := "test-cached-" + cacheKey("test-cached", other)

	// no cache: startup fails
	_, err := NewCachedFromConfig(context.TODO(), "test-cached", other)
	require.Error(t, err)

	// outdated cache: startup succeeds, cached rates served
	require.NoError(t, cache.Put(key, &cached{
		Type:    api.TariffTypePriceForecast,
		Rates:   stale,
		Updated: now.Add(-time.Hour),
	}))

	res, err := NewCachedFromConfig(context.TODO(), "test-cached", other)
	require.NoError(t, err)

	rr, err := res.Rates()
	require.NoError(t, err)
	assert.Equal(t, stale, rr)
	assert.Equal(t, api.TariffTypePriceForecast, res.Type())

	// tariff becomes available but retry interval not elapsed: cached rates served
	tariff.setErr(nil)
	rr, err = res.Rates()
	require.NoError(t, err)
	assert.Equal(t, stale, rr)

	// retry interval elapsed: next Rates() call creates the tariff in the background
	p := res.(*cachingProxy)
	p.mu.Lock()
	p.retryAt = time.Time{}
	p.mu.Unlock()

	require.Eventually(t, func() bool {
		rr, err := res.Rates()
		return err == nil && rr[0].Value == live[0].Value
	}, time.Second, 10*time.Millisecond)

	// tariff becomes unavailable at runtime: last rates served
	tariff.setErr(api.ErrOutdated)
	rr, err = res.Rates()
	require.NoError(t, err)
	assert.Equal(t, live, rr)
}

func TestCacheInterval(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	p := &cachingProxy{key: "test", interval: time.Hour}
	require.NoError(t, p.cachePut(api.TariffTypeSolar, api.Rates{
		{Start: time.Now(), End: time.Now().Add(time.Hour), Value: 1},
	}))

	// within interval
	res, err := p.cacheGet()
	require.NoError(t, err)
	require.Equal(t, api.TariffTypeSolar, res.Type)

	// interval elapsed
	p.cached.Updated = time.Now().Add(-2 * time.Hour)
	_, err = p.cacheGet()
	require.Error(t, err)
}
