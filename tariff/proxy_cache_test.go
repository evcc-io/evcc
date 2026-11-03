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
	stale := makeRates(now.Add(-time.Hour), SlotDuration, 8, 0)
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

func TestCachedFallbackElapsed(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	other := map[string]any{"interval": 60 * time.Millisecond}
	key := "test-retry-" + cacheKey("test-retry", other)

	// cached rates have entirely elapsed
	now := time.Now().UTC().Truncate(SlotDuration)
	elapsed := makeRates(now.Add(-2*time.Hour), SlotDuration, 4, 0)
	require.NoError(t, cache.Put(key, &cached{
		Type:    api.TariffTypePriceForecast,
		Rates:   elapsed,
		Updated: now.Add(-2 * time.Hour),
	}))

	// startup fails rather than reporting an available tariff without usable rates
	retryable.setErr(errors.New("unavailable"))
	_, err := NewCachedFromConfig(context.TODO(), "test-retry", other)
	require.Error(t, err)

	// tariff available at startup, then unavailable at runtime with elapsed cache only
	retryable.setErr(nil)
	res, err := NewCachedFromConfig(context.TODO(), "test-retry", other)
	require.NoError(t, err)

	retryable.setErr(api.ErrOutdated)
	_, err = res.Rates()
	require.Error(t, err)
}

func TestCachedFreshElapsed(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	other := map[string]any{"interval": time.Hour}
	key := "test-retry-" + cacheKey("test-retry", other)

	// cache updated within the interval, but its rates have entirely elapsed
	now := time.Now().UTC().Truncate(SlotDuration)
	putElapsed := func() {
		require.NoError(t, cache.Put(key, &cached{
			Type:    api.TariffTypePriceForecast,
			Rates:   makeRates(now.Add(-2*time.Hour), SlotDuration, 4, 0),
			Updated: now,
		}))
	}

	// the tariff is created instead of serving the elapsed rates
	putElapsed()
	retryable.setErr(nil)
	res, err := NewCachedFromConfig(context.TODO(), "test-retry", other)
	require.NoError(t, err)

	rr, err := res.Rates()
	require.NoError(t, err)
	require.NotEmpty(t, rr)
	assert.True(t, rr[len(rr)-1].End.After(time.Now()))

	// without a tariff to create, the elapsed rates are not served either
	putElapsed()
	retryable.setErr(errors.New("unavailable"))
	_, err = NewCachedFromConfig(context.TODO(), "test-retry", other)
	require.Error(t, err)
}

func TestCacheKeepsLeadingSlots(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	now := time.Now().Truncate(SlotDuration)

	// the fresh tariff's first fetch starts an hour out, the cache covers the hour before
	p := &cachingProxy{
		key:      "test-merge",
		interval: time.Hour,
		tariff:   &testTariff{typ: api.TariffTypeCo2, rates: makeRates(now.Add(time.Hour), SlotDuration, 4, 10)},
		cached:   &cached{Type: api.TariffTypeCo2, Rates: makeRates(now.Add(-time.Hour), SlotDuration, 12, 0), Updated: now.Add(-2 * time.Hour)},
	}

	rr, err := p.Rates()
	require.NoError(t, err)

	// elapsed cached slots dropped, the current hour kept, fresh rates appended
	assert.Equal(t, append(makeRates(now, SlotDuration, 4, 4), makeRates(now.Add(time.Hour), SlotDuration, 4, 10)...), rr)
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
