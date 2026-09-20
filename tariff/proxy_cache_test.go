package tariff

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/db/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCachedFallback(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	other := map[string]any{"interval": 50 * time.Millisecond}
	key := "test-retry-" + cacheKey("test-retry", other)

	// no cache: startup fails
	retryable.setErr(errors.New("unavailable"))
	_, err := NewCachedFromConfig(context.TODO(), "test-retry", other)
	require.Error(t, err)

	// outdated cache: startup succeeds, cached rates served
	// utc for comparing cached rates after json round trip
	now := time.Now().UTC().Truncate(SlotDuration)
	stale := makeRates(now.Add(-time.Hour), SlotDuration, 4, 0)
	require.NoError(t, cache.Put(key, &cached{
		Type:    api.TariffTypePriceForecast,
		Rates:   stale,
		Updated: now.Add(-time.Hour),
	}))

	res, err := NewCachedFromConfig(context.TODO(), "test-retry", other)
	require.NoError(t, err)

	rr, err := res.Rates()
	require.NoError(t, err)
	assert.Equal(t, stale, rr)
	assert.Equal(t, api.TariffTypePriceForecast, res.Type())

	// tariff becomes available: wrapper retry creates it, live rates served and cached
	retryable.setErr(nil)
	w := res.(*cachingProxy).tariff.(*Wrapper)
	w.mu.Lock()
	w.retriedAt = time.Time{}
	w.mu.Unlock()

	live, err := res.Rates()
	require.NoError(t, err)
	assert.NotEqual(t, stale, live)

	// tariff becomes unavailable at runtime: last rates served
	retryable.setErr(api.ErrOutdated)
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
