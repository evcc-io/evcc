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
	err    error // last instantiation error, retry is running while set
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
	if _, err := p.cacheGet(); err != nil {
		// attempt to create a new instance
		if err := p.createInstance(); err != nil {
			// if no cached data available, return error
			if !p.hasCache() {
				return nil, err
			}

			// use outdated cached data until tariff becomes available
			p.log.WARN.Printf("tariff not available, using cached rates (updated: %s): %v", p.cached.Updated.Local(), err)
		}
	}

	if p.tariff == nil {
		p.log.DEBUG.Printf("using cache: %s (updated: %s)", p.key, p.cached.Updated.Local())
	}

	return p, nil
}

// createInstance creates the tariff. If that fails and cached rates are available, it keeps retrying in the background.
func (p *cachingProxy) createInstance() error {
	t, err := NewFromConfig(p.ctx, p.typ, p.config)
	if err != nil {
		p.err = err
		if p.hasCache() {
			go p.retry()
		}
		return err
	}

	p.tariff = t
	return nil
}

// retry creates the tariff at regular interval until it becomes available
func (p *cachingProxy) retry() {
	interval := p.interval
	if interval <= 0 {
		interval = defaultInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
		}

		t, err := NewFromConfig(p.ctx, p.typ, p.config)
		if err != nil {
			p.log.WARN.Printf("tariff not available, using cached rates: %v", err)
			continue
		}

		p.mu.Lock()
		p.tariff = t
		p.err = nil
		p.mu.Unlock()

		p.log.INFO.Printf("tariff available: %s", p.key)
		return
	}
}

// instance returns the tariff, creating it once cached data is outdated. Returns nil if unavailable.
func (p *cachingProxy) instance() api.Tariff {
	if p.tariff == nil && p.err == nil {
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

	err := p.err
	if t := p.instance(); t != nil {
		var res api.Rates
		if res, err = t.Rates(); err == nil {
			if p.dynamicTariff() {
				err = p.cachePut(t.Type(), res)
			}
			return res, err
		}
		// tariff keeps updating itself, serve cached rates until it recovers
	}

	if p.hasCache() {
		return slices.Clone(p.cached.Rates), nil
	}

	return nil, err
}

// Type returns the tariff type
func (p *cachingProxy) Type() api.TariffType {
	p.mu.Lock()
	defer p.mu.Unlock()

	if t := p.instance(); t != nil {
		return t.Type()
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

// cacheGet returns cached data if the update interval has not yet elapsed
func (p *cachingProxy) cacheGet() (*cached, error) {
	if p.cached == nil {
		res, err := cacheGet(p.key)
		if err != nil {
			return nil, err
		}

		p.cached = res
	}

	if !p.hasCache() {
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
