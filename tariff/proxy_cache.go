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
	"github.com/evcc-io/evcc/db/cache"
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

	log := util.NewLogger("tariff")

	// check if cached data is up to date
	if _, err := p.cacheGet(); err != nil {
		// attempt to create a new instance
		if err := p.createInstance(); err != nil {
			// if no usable cached data available, return error
			if !p.usableCache() {
				return nil, err
			}

			// use outdated cached data until tariff becomes available
			log.WARN.Printf("tariff not available, using cached rates (updated: %s): %v", p.cached.Updated.Local(), err)
		}
	}

	if p.tariff == nil {
		log.DEBUG.Printf("using cache: %s (updated: %s)", p.key, p.cached.Updated.Local())
	}

	return p, nil
}

// createInstance creates the tariff. If that fails, the wrapper takes its place.
func (p *cachingProxy) createInstance() error {
	t, err := NewFromConfig(p.ctx, p.typ, p.config)
	if err != nil {
		t = NewWrapper(p.ctx, p.typ, p.config, err)
	}

	p.tariff = t
	return err
}

// instance returns the tariff, creating it once cached data is outdated
func (p *cachingProxy) instance() api.Tariff {
	if p.tariff == nil {
		if _, err := p.cacheGet(); err != nil {
			_ = p.createInstance()
		}
	}

	return p.tariff
}

// Rates returns cached data until underlying tariff is created, then delegates to tariff.
// Cached data is served while the tariff is unavailable.
func (p *cachingProxy) Rates() (api.Rates, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	t := p.instance()
	if t == nil {
		// cached data is up to date
		return slices.Clone(p.cached.Rates), nil
	}

	res, err := t.Rates()
	if err != nil {
		if p.usableCache() {
			return slices.Clone(p.cached.Rates), nil
		}
		return nil, err
	}

	// a fresh tariff starts with its first fetch, keep the cached slots leading up to it
	if p.hasCache() {
		res = mergeAfter(p.cached.Rates, res, time.Now().Truncate(SlotDuration))
	}

	if p.dynamicTariff() {
		err = p.cachePut(t.Type(), res)
	}

	return res, err
}

// Type returns the tariff type
func (p *cachingProxy) Type() api.TariffType {
	p.mu.Lock()
	defer p.mu.Unlock()

	if t := p.instance(); t != nil {
		// wrapper reports unknown type while tariff is unavailable
		if typ := t.Type(); typ != 0 {
			return typ
		}
	}

	if p.hasCache() {
		return p.cached.Type
	}

	return 0 // unknown
}

func (p *cachingProxy) dynamicTariff() bool {
	return slices.Contains([]api.TariffType{
		api.TariffTypePriceForecast,
		api.TariffTypeCo2,
		api.TariffTypeSolar,
	}, p.tariff.Type())
}

// hasCache returns true if cached rates are available, regardless of their age
func (p *cachingProxy) hasCache() bool {
	return p.cached != nil && len(p.cached.Rates) > 0
}

// usableCache returns true if cached rates still reach into the future. Rates are sorted by start,
// so the last slot ends latest. Entirely elapsed rates cannot inform any decision.
func (p *cachingProxy) usableCache() bool {
	return p.hasCache() && p.cached.Rates[len(p.cached.Rates)-1].End.After(time.Now())
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

	if !p.usableCache() {
		return nil, errors.New("no rates")
	}

	if d := time.Since(p.cached.Updated); d > p.interval {
		return nil, fmt.Errorf("cache outdated: %v", d.Round(time.Second))
	}

	return p.cached, nil
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
		Rates:   rates,
		Updated: p.updated,
	}

	return cache.Put(p.key, p.cached)
}
