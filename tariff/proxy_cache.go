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

// defaultInterval is the update interval assumed if the tariff doesn't configure one
const defaultInterval = time.Hour

// cachingProxy wraps a tariff with caching
type cachingProxy struct {
	mu   sync.Mutex
	hash [32]byte

	key      string
	ctx      context.Context
	typ      string
	config   map[string]any
	interval time.Duration
	updated  time.Time

	cached *cached
	tariff api.Tariff

	creating bool      // instance creation in progress
	retryAt  time.Time // earliest next instance creation attempt
	err      error     // last instance creation error
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
		ctx:      ctx,
		typ:      typ,
		config:   other,
		interval: cc.Interval,
		key:      tariffType + "-" + cacheKey(typ, other),
	}

	// check if cached data is up to date
	data, err := p.cacheGet()
	if err != nil {
		// attempt to create a new instance
		tariff, err := NewFromConfig(ctx, typ, other)
		if err != nil {
			// if no cached data available, return error
			if p.cached == nil {
				return nil, err
			}

			// use outdated cached data
			data = p.cached
			p.creationFailed(err)
		}

		// if instance creation was successful, use it, otherwise use outdated cached data
		if err == nil {
			p.tariff = tariff
		}
	}

	if data != nil {
		log := util.NewLogger("tariff")
		log.DEBUG.Printf("using cache: %s (updated: %s)", p.key, data.Updated.Local())
	}

	return p, nil
}

func (p *cachingProxy) creationFailed(err error) {
	p.err = err
	p.retryAt = time.Now().Add(p.interval)
}

// createInstance creates the tariff in the background, retrying at most once per interval
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
			util.NewLogger("tariff").ERROR.Printf("creating tariff failed: %v", err)
			p.creationFailed(err)
			return
		}

		p.tariff = t
		p.err = nil
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

// staleRates returns cached data regardless of its age as long as it extends into the future
func (p *cachingProxy) staleRates(err error) (api.Rates, error) {
	if err == nil {
		err = api.ErrNotAvailable
	}

	if p.cached == nil {
		return nil, err
	}

	now := time.Now()
	if !slices.ContainsFunc(p.cached.Rates, func(r api.Rate) bool {
		return r.End.After(now)
	}) {
		return nil, err
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
