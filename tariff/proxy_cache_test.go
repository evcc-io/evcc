package tariff

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/db/cache"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheInterval(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	p := &cachingProxy{key: "test", interval: time.Hour, log: util.NewLogger("tariff")}
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

// cacheTestTariff is a forecast tariff whose upstream availability is controlled by the test
type cacheTestTariff struct{}

func (t *cacheTestTariff) Rates() (api.Rates, error) {
	return cacheTestRates()
}

func (t *cacheTestTariff) Type() api.TariffType {
	return api.TariffTypePriceForecast
}

var (
	cacheTestDown  atomic.Bool
	cacheTestRates func() (api.Rates, error)
)

func init() {
	registry.Add("cachetest", func(map[string]any) (api.Tariff, error) {
		if cacheTestDown.Load() {
			return nil, errors.New("api down")
		}
		return new(cacheTestTariff), nil
	})
}

func testRates(value float64) api.Rates {
	now := time.Now().Truncate(SlotDuration)
	return api.Rates{{Start: now, End: now.Add(SlotDuration), Value: value}}
}

// assertRateValue checks the rates value, ignoring time zone differences from json round trips
func assertRateValue(t *testing.T, rr api.Rates, value float64, msgAndArgs ...any) {
	t.Helper()
	require.Len(t, rr, 1, msgAndArgs...)
	assert.Equal(t, value, rr[0].Value, msgAndArgs...)
}

func newCacheTestProxy(t *testing.T, cfg map[string]any) *cachingProxy {
	t.Helper()
	p, err := NewCachedFromConfig(t.Context(), "cachetest", cfg)
	require.NoError(t, err)
	return p.(*cachingProxy)
}

// ageCache backdates the persisted cache entry beyond the update interval
func ageCache(t *testing.T, p *cachingProxy) {
	t.Helper()
	p.cached.Updated = time.Now().Add(-2 * p.interval)
	require.NoError(t, cache.Put(p.key, p.cached))
}

func waitForInstance(t *testing.T, p *cachingProxy) {
	t.Helper()
	require.Eventually(t, func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return !p.creating
	}, time.Second, time.Millisecond)
}

func TestCacheFreshAtStartup(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	cfg := map[string]any{"interval": "1h"}

	cacheTestDown.Store(false)
	cacheTestRates = func() (api.Rates, error) { return testRates(1), nil }
	p := newCacheTestProxy(t, cfg)
	_, err := p.Rates()
	require.NoError(t, err)

	// restart within interval with api down: cache served without contacting upstream
	cacheTestDown.Store(true)
	p = newCacheTestProxy(t, cfg)
	rr, err := p.Rates()
	require.NoError(t, err)
	assertRateValue(t, rr, 1)
	assert.Equal(t, api.TariffTypePriceForecast, p.Type())
	assert.Nil(t, p.tariff)
}

func TestCacheOutdatedAtStartup(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	cfg := map[string]any{"interval": "1h"}

	cacheTestDown.Store(false)
	cacheTestRates = func() (api.Rates, error) { return testRates(1), nil }
	p := newCacheTestProxy(t, cfg)
	_, err := p.Rates()
	require.NoError(t, err)
	ageCache(t, p)

	// restart after interval with api down: outdated cache served as fallback
	cacheTestDown.Store(true)
	p = newCacheTestProxy(t, cfg)
	waitForInstance(t, p)
	require.Nil(t, p.tariff)
	require.Error(t, p.err)

	rr, err := p.Rates()
	require.NoError(t, err)
	assertRateValue(t, rr, 1)
	assert.Equal(t, api.TariffTypePriceForecast, p.Type())

	// api recovers: next retry creates the instance and live data replaces the cache
	cacheTestDown.Store(false)
	cacheTestRates = func() (api.Rates, error) { return testRates(2), nil }
	p.mu.Lock()
	p.retryAt = time.Time{}
	p.mu.Unlock()

	rr, err = p.Rates()
	require.NoError(t, err)
	assertRateValue(t, rr, 1, "cache served while instance is created")
	waitForInstance(t, p)

	rr, err = p.Rates()
	require.NoError(t, err)
	assertRateValue(t, rr, 2)
	require.NoError(t, p.err)
}

func TestCacheRetryInterval(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	cfg := map[string]any{"interval": "1h"}

	cacheTestDown.Store(false)
	cacheTestRates = func() (api.Rates, error) { return testRates(1), nil }
	p := newCacheTestProxy(t, cfg)
	_, err := p.Rates()
	require.NoError(t, err)
	ageCache(t, p)

	cacheTestDown.Store(true)
	p = newCacheTestProxy(t, cfg)
	waitForInstance(t, p)

	// the first retry is scheduled well before the update interval
	assert.Equal(t, minRetryDelay, p.retryDelay)
	assert.Less(t, time.Until(p.retryAt), p.interval)

	// api recovers but the retry delay has not elapsed: keep serving cache
	cacheTestDown.Store(false)
	_, err = p.Rates()
	require.NoError(t, err)
	waitForInstance(t, p)
	assert.Nil(t, p.tariff)

	// repeated failure backs off up to the update interval
	p.mu.Lock()
	for range 10 {
		p.creationFailed(errors.New("api down"))
	}
	delay := p.retryDelay
	p.mu.Unlock()
	assert.Equal(t, p.interval, delay)
}

func TestCacheFallbackAge(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	cfg := map[string]any{"interval": "1h"}

	cacheTestDown.Store(false)
	cacheTestRates = func() (api.Rates, error) { return testRates(1), nil }
	p := newCacheTestProxy(t, cfg)
	_, err := p.Rates()
	require.NoError(t, err)

	// rates still reach into the future but the cache is older than the fallback limit
	now := time.Now().Truncate(SlotDuration)
	require.NoError(t, cache.Put(p.key, &cached{
		Type:    api.TariffTypePriceForecast,
		Rates:   api.Rates{{Start: now, End: now.Add(SlotDuration), Value: 1}},
		Updated: time.Now().Add(-maxFallbackAge - time.Minute),
	}))

	cacheTestDown.Store(true)
	_, err = NewCachedFromConfig(t.Context(), "cachetest", cfg)
	require.Error(t, err, "outdated cache must not mask an unavailable provider")
}

func TestCacheRuntimeOutage(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	cacheTestDown.Store(false)
	cacheTestRates = func() (api.Rates, error) { return testRates(1), nil }
	p := newCacheTestProxy(t, map[string]any{"interval": "1h"})
	require.NotNil(t, p.tariff)

	rr, err := p.Rates()
	require.NoError(t, err)
	assertRateValue(t, rr, 1)

	// tariff data outdated: last known rates served from cache and flagged as stale
	cacheTestRates = func() (api.Rates, error) { return nil, api.ErrOutdated }
	rr, err = p.Rates()
	require.NoError(t, err)
	assertRateValue(t, rr, 1)
	assert.True(t, p.stale)

	// tariff recovers
	cacheTestRates = func() (api.Rates, error) { return testRates(2), nil }
	rr, err = p.Rates()
	require.NoError(t, err)
	assertRateValue(t, rr, 2)
	assert.False(t, p.stale)
}

// expiredRates returns rates that have entirely elapsed
func expiredRates() api.Rates {
	now := time.Now().Truncate(SlotDuration)
	return api.Rates{{Start: now.Add(-2 * time.Hour), End: now.Add(-time.Hour), Value: 1}}
}

// expireCache replaces the persisted entry with outdated, fully elapsed rates
func expireCache(t *testing.T, p *cachingProxy) {
	t.Helper()
	require.NoError(t, cache.Put(p.key, &cached{
		Type:    api.TariffTypePriceForecast,
		Rates:   expiredRates(),
		Updated: time.Now().Add(-2 * p.interval),
	}))
}

func TestCacheExpiredAtStartup(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	cfg := map[string]any{"interval": "1h"}

	cacheTestDown.Store(false)
	cacheTestRates = func() (api.Rates, error) { return testRates(1), nil }
	p := newCacheTestProxy(t, cfg)
	_, err := p.Rates()
	require.NoError(t, err)

	// long shutdown: cache is outdated and holds nothing that could be served
	expireCache(t, p)

	// the instance is created upfront so the first call returns the provider's rates
	p = newCacheTestProxy(t, cfg)
	require.NotNil(t, p.tariff)

	rr, err := p.Rates()
	require.NoError(t, err)
	assertRateValue(t, rr, 1)
}

func TestCacheExpiredAtStartupUnavailable(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	cfg := map[string]any{"interval": "1h"}

	cacheTestDown.Store(false)
	cacheTestRates = func() (api.Rates, error) { return testRates(1), nil }
	p := newCacheTestProxy(t, cfg)
	_, err := p.Rates()
	require.NoError(t, err)

	expireCache(t, p)

	// unusable cache and failing provider surface the error instead of a silent stub
	cacheTestDown.Store(true)
	_, err = NewCachedFromConfig(t.Context(), "cachetest", cfg)
	require.Error(t, err)
}

func TestCacheExpiredRates(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	cacheTestDown.Store(false)
	cacheTestRates = func() (api.Rates, error) { return testRates(1), nil }
	p := newCacheTestProxy(t, map[string]any{"interval": "1h"})
	_, err := p.Rates()
	require.NoError(t, err)

	// cached rates entirely in the past are not served
	p.cached.Rates = api.Rates{{Start: time.Now().Add(-2 * time.Hour), End: time.Now().Add(-time.Hour), Value: 1}}
	cacheTestRates = func() (api.Rates, error) { return nil, api.ErrOutdated }
	_, err = p.Rates()
	require.ErrorIs(t, err, api.ErrOutdated)
}
