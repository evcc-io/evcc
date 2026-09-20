package tariff

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const (
	// defaultInterval is the update interval assumed if the tariff doesn't configure one
	defaultInterval = time.Hour

	// maxFallbackAge limits how long cached rates are served while the tariff is unavailable.
	// Beyond that the provider is considered broken rather than briefly unreachable, and
	// callers fall back to their no-rates behaviour instead of planning on outdated data.
	maxFallbackAge = 24 * time.Hour

	// minRetryDelay is the initial delay before instance creation is retried. It doubles on
	// repeated failure up to the update interval, which bounds requests against the provider.
	minRetryDelay = time.Minute
)

// cachingProxy wraps a tariff with caching
type cachingProxy struct {
	mu   sync.Mutex
	log  *util.Logger
	hash [32]byte

	key      string
	ctx      context.Context
	typ      string
	config   map[string]any
	interval time.Duration
	updated  time.Time

	cached *cached
	tariff api.Tariff

	creating   bool          // instance creation in progress
	retryAt    time.Time     // earliest next instance creation attempt
	retryDelay time.Duration // delay before the next attempt
	err        error         // last instance creation error
	stale      bool          // cached rates are currently served as fallback
}

var _ api.Tariff = (*cachingProxy)(nil)

// NewCachedFromConfig creates a proxy that controls tariff instantiation and caching
func NewCachedFromConfig(ctx context.Context, typ string, other map[string]any) (api.Tariff, error) {
	tariffType := typ
	if template := util.TemplateName(typ, other); template != "" {
		tariffType = template
	}

	cc := struct {
		Interval time.Duration
		Other    map[string]any `mapstructure:",remain"`
	}{
		Interval: defaultInterval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	p := &cachingProxy{
		log:      util.NewLogger("tariff"),
		ctx:      ctx,
		typ:      typ,
		config:   other,
		interval: cc.Interval,
		key:      tariffType + "-" + cacheKey(typ, other),
	}

	// check if cached data is up to date
	data, err := p.cacheGet()
	if err != nil {
		// without rates to fall back on the instance must be created upfront,
		// otherwise Rates would report the tariff as unavailable until it is ready
		if !p.fallbackAvailable() {
			tariff, err := NewFromConfig(ctx, typ, other)
			if err != nil {
				return nil, err
			}

			p.tariff = tariff
		} else {
			// use outdated cached data until instance is created in the background
			data = p.cached
			p.createInstance()
		}
	}

	if data != nil {
		p.log.DEBUG.Printf("using cache: %s (updated: %s)", p.key, data.Updated.Local())
	}

	return p, nil
}

// creationFailed records a failed attempt and schedules the next one
func (p *cachingProxy) creationFailed(err error) {
	p.err = err

	if p.retryDelay == 0 {
		p.retryDelay = minRetryDelay
	} else {
		p.retryDelay *= 2
	}
	p.retryDelay = min(p.retryDelay, p.interval)

	p.retryAt = time.Now().Add(p.retryDelay)
}

// createInstance creates the tariff in the background. Recovery is driven by Rates,
// which the site loop calls regularly.
func (p *cachingProxy) createInstance() {
	if p.creating || time.Now().Before(p.retryAt) {
		return
	}

	p.creating = true

	go func() {
		t, err := NewFromConfig(p.ctx, p.typ, p.config)

		p.mu.Lock()
		defer p.mu.Unlock()

		p.creating = false

		if err != nil {
			p.log.ERROR.Printf("creating tariff failed: %v", err)
			p.creationFailed(err)
			return
		}

		p.tariff = t
		p.err = nil
		p.retryDelay = 0
	}()
}

// Rates returns cached data until underlying tariff is created, then delegates to tariff.
// Cached data is served as fallback while the tariff is unavailable.
func (p *cachingProxy) Rates() (api.Rates, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.tariff == nil {
		if res, err := p.cacheGet(); err == nil {
			return slices.Clone(res.Rates), nil
		}

		p.createInstance()

		return p.staleRates(p.err)
	}

	res, err := p.tariff.Rates()
	if err != nil {
		return p.staleRates(err)
	}

	if p.stale {
		p.stale = false
		p.log.INFO.Printf("%s recovered", p.typ)
	}

	if p.dynamicTariff() {
		err = p.cachePut(p.tariff.Type(), res)
	}

	return res, err
}

// Type returns the tariff type
func (p *cachingProxy) Type() api.TariffType {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.tariff == nil {
		if p.cached != nil {
			return p.cached.Type
		}

		return 0
	}

	return p.tariff.Type()
}

func (p *cachingProxy) dynamicTariff() bool {
	return slices.Contains([]api.TariffType{
		api.TariffTypePriceForecast,
		api.TariffTypeCo2,
		api.TariffTypeSolar,
	}, p.tariff.Type())
}

// cacheGet returns cached data if the update interval has not yet elapsed
func (p *cachingProxy) cacheGet() (*cached, error) {
	if p.cached == nil {
		res, err := cacheGet(p.key)
		if err != nil {
			return nil, err
		}

		p.cached = res
	}

	if len(p.cached.Rates) == 0 {
		return nil, errors.New("no rates")
	}

	if d := time.Since(p.cached.Updated); d > p.interval {
		return nil, fmt.Errorf("cache outdated: %v", d.Round(time.Second))
	}

	return p.cached, nil
}

// fallbackAvailable indicates cached rates that may be served while the tariff is unavailable
func (p *cachingProxy) fallbackAvailable() bool {
	if p.cached == nil || time.Since(p.cached.Updated) > maxFallbackAge {
		return false
	}

	now := time.Now()
	return slices.ContainsFunc(p.cached.Rates, func(r api.Rate) bool {
		return r.End.After(now)
	})
}

// staleRates returns cached data beyond the update interval as long as it remains usable
func (p *cachingProxy) staleRates(err error) (api.Rates, error) {
	if !p.fallbackAvailable() {
		if err == nil {
			err = api.ErrNotAvailable
		}
		return nil, err
	}

	if !p.stale {
		p.stale = true
		p.log.WARN.Printf("%s unavailable, using rates cached at %s: %v", p.typ, p.cached.Updated.Local(), err)
	}

	return slices.Clone(p.cached.Rates), nil
}

// cachePut persists rates if changed or the update interval has elapsed
func (p *cachingProxy) cachePut(typ api.TariffType, rates api.Rates) error {
	hash := sha256.Sum256(fmt.Append(nil, rates))
	if hash == p.hash && time.Since(p.updated) < p.interval {
		return nil
	}

	p.hash = hash
	p.updated = time.Now()
	p.cached = &cached{
		Type:    typ,
		Rates:   slices.Clone(rates),
		Updated: p.updated,
	}

	return cachePut(p.key, typ, rates)
}
