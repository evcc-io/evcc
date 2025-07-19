package tariff

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/jinzhu/now"
)

// CachingProxy wraps a tariff with caching
type CachingProxy struct {
	mu sync.Mutex

	key    string
	ctx    context.Context
	typ    string
	config map[string]any

	tariff api.Tariff
}

var _ api.Tariff = (*CachingProxy)(nil)

// NewCachedFromConfig creates a proxy that controls tariff instantiation and caching
func NewCachedFromConfig(ctx context.Context, typ string, other map[string]any) (api.Tariff, error) {
	tariffType := typ
	if typ == "template" {
		if template, ok := other["template"].(string); ok {
			tariffType = template
		}
	}

	p := &CachingProxy{
		ctx:    ctx,
		typ:    typ,
		config: other,
		key:    tariffType + "-" + cacheKey(typ, other),
	}

	// not cached yet, create instance
	if data, err := p.cacheGet(); err != nil {
		tariff, err := NewFromConfig(ctx, typ, other)
		if err != nil {
			return nil, err
		}

		p.tariff = tariff
	} else {
		log := util.NewLogger("tariff")
		log.DEBUG.Printf("using cache: %s (start: %s, end: %s)", p.key,
			data.Rates[0].Start.Local(), data.Rates[len(data.Rates)-1].End.Local(),
		)
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
		err = p.cachePut(p.tariff.Type(), res)
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

	// consider cache valid if it contains rates for 24 hours
	until := time.Now().Add(24 * time.Hour)

	if !ratesValid(res.Rates, until) {
		return nil, errors.New("not enough rates")
	}

	res.Rates = currentRates(res.Rates)
	if len(res.Rates) == 0 {
		return nil, errors.New("no current rates")
	}

	return res, nil
}

func (p *CachingProxy) cachePut(typ api.TariffType, rates api.Rates) error {
	return cachePut(p.key, typ, rates)
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
