package tariff

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/jinzhu/now"
)

// CachingProxy wraps a tariff with caching
type CachingProxy struct {
	// log *util.Logger
	mu sync.Mutex

	key    string
	ctx    context.Context
	typ    string
	config map[string]any

	tariff api.Tariff // Will be nil initially
}

var _ api.Tariff = (*CachingProxy)(nil)

// NewCachedFromConfig creates a proxy that controls tariff instantiation and caching
func NewCachedFromConfig(ctx context.Context, typ string, other map[string]any) (api.Tariff, error) {
	// actualTyp := typ
	// if typ == "template" {
	// 	if template, ok := other["template"].(string); ok {
	// 		actualTyp = template
	// 	}
	// }

	// cache, hash := NewSolarCacheManager(actualTyp, other)
	// log := util.NewLogger(fmt.Sprintf("%s-%s", actualTyp, "hash"))

	p := &CachingProxy{
		// log:    log,
		ctx:    ctx,
		typ:    typ,
		config: other,
		key:    cacheKey(typ, other),
	}

	// not cached yet, create instance
	if _, err := p.cacheGet(); err != nil {
		tariff, err := NewFromConfig(ctx, typ, other)
		if err != nil {
			return nil, err
		}

		p.tariff = tariff
	}

	return p, nil
}

func (p *CachingProxy) createInstance() {
	t, err := NewFromConfig(p.ctx, p.typ, p.config)
	if err != nil {
		t = &proxyError{err}
	}

	p.tariff = t
}

// Rates returns cached data until underlying tariff is created, then delegates to tariff
func (p *CachingProxy) Rates() (api.Rates, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.tariff == nil {
		if res, err := p.cacheGet(); err == nil {
			return res.Rates, nil
		}

		p.createInstance()
	}

	res, err := p.tariff.Rates()
	if err != nil {
		return nil, err
	}

	if p.dynamicTariff() {
		err = p.cachePut(res)
	}

	return res, err
}

// Type returns the tariff type
func (p *CachingProxy) Type() api.TariffType {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.tariff == nil {
		if res, err := p.cacheGet(); err == nil {
			return res.Type
		}

		p.createInstance()
	}

	return p.tariff.Type()
}

func (p *CachingProxy) dynamicTariff() bool {
	return slices.Contains([]api.TariffType{
		api.TariffTypePriceForecast,
		api.TariffTypeCo2,
		api.TariffTypeSolar,
	}, p.tariff.Type())
}

func (p *CachingProxy) cacheGet() (*cached, error) {
	res, err := cacheGet(p.key)
	if err != nil {
		return nil, err
	}

	// consider cache valid if it contains rates for today
	until := now.With(time.Now()).EndOfDay().Add(time.Nanosecond) // add ns for half-closed interval

	if !ratesValid(res.Rates, until) {
		return res, nil
	}

	res.Rates = currentRates(res.Rates)

	return res, nil
}

func (p *CachingProxy) cachePut(rates api.Rates) error {
	return cachePut(p.key, p.Type(), rates)
}

func ratesValid(rr api.Rates, until time.Time) bool {
	if len(rr) == 0 {
		return false
	}

	rr.Sort()

	return !rr[len(rr)-1].End.Before(until)
}

func currentRates(rr api.Rates) api.Rates {
	res := make(api.Rates, 0, len(rr))
	now := now.With(time.Now()).BeginningOfHour()

	for _, r := range rr {
		if !r.End.Before(now) {
			res = append(res, r)
		}
	}

	return res
}
