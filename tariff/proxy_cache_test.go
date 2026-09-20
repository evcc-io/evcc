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

func TestCachedFallback(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	// utc for comparing cached rates after json round trip
	now := time.Now().UTC().Truncate(SlotDuration)
	live := makeRates(now, SlotDuration, 4, 10)
	stale := makeRates(now.Add(-time.Hour), SlotDuration, 4, 0)

	flaky.mu.Lock()
	flaky.rates = live
	flaky.err = errors.New("unavailable")
	flaky.mu.Unlock()

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

	// tariff becomes available: wrapper retry creates it, live rates served and cached
	flaky.setErr(nil)
	w := res.(*cachingProxy).tariff.(*Wrapper)
	w.mu.Lock()
	w.retriedAt = time.Time{}
	w.mu.Unlock()

	rr, err = res.Rates()
	require.NoError(t, err)
	assert.Equal(t, live, rr)

	// tariff becomes unavailable at runtime: last rates served
	flaky.setErr(api.ErrOutdated)
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
