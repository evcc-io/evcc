package tariff

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/db/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	// api recovers but retry interval has not elapsed: keep serving cache
	cacheTestDown.Store(false)
	_, err = p.Rates()
	require.NoError(t, err)
	waitForInstance(t, p)
	assert.Nil(t, p.tariff)
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

	// tariff data outdated: last known rates served from cache
	cacheTestRates = func() (api.Rates, error) { return nil, api.ErrOutdated }
	rr, err = p.Rates()
	require.NoError(t, err)
	assertRateValue(t, rr, 1)

	// tariff recovers
	cacheTestRates = func() (api.Rates, error) { return testRates(2), nil }
	rr, err = p.Rates()
	require.NoError(t, err)
	assertRateValue(t, rr, 2)
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
