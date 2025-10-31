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
	"github.com/jinzhu/now"
)

// cachingProxy wraps a tariff with caching
type cachingProxy struct {
	mu   sync.Mutex
	hash [32]byte

	key    string
	ctx    context.Context
	typ    string
	config map[string]any

	cached *cached
	tariff api.Tariff
}

var _ api.Tariff = (*cachingProxy)(nil)

// NewCachedFromConfig creates a proxy that controls tariff instantiation and caching
func NewCachedFromConfig(ctx context.Context, typ string, other map[string]any) (api.Tariff, error) {
	tariffType := typ
	if typ == "template" {
		if template, ok := other["template"].(string); ok {
			tariffType = template
		}
	}

	p := &cachingProxy{
		ctx:    ctx,
		typ:    typ,
		config: other,
		key:    tariffType + "-" + cacheKey(typ, other),
	}

	// check if we have cached data until end of tomorrow
	data, err := p.cacheGet(untilEndOfTomorrow())
	if err != nil {
		// attempt to create a new instance
		tariff, err := NewFromConfig(ctx, typ, other)
		if err != nil {
			// check if we have at least data for the next 24 hours
			atLeast2hrs, err2 := p.cacheGet(for24hrs())
			if err2 != nil {
				// if not available, return error
				return nil, err
			}

			// use cached data for the next 24 hours
			data = atLeast2hrs
		}

		// if instance creation was successful, cache it, otherwise use cached 24hrs of data
		if err == nil {
			p.tariff = tariff
		}
	}

	if data != nil {
		log := util.NewLogger("tariff")
		log.DEBUG.Printf("using cache: %s (start: %s, end: %s)", p.key,
			data.Rates[0].Start.Local(), data.Rates[len(data.Rates)-1].End.Local(),
		)
	}

	return p, nil
}

func (p *cachingProxy) createInstance() {
	t, err := NewFromConfig(p.ctx, p.typ, p.config)
	if err != nil {
		t = &proxyError{err}
	}

	p.tariff = t
}

// Rates returns cached data until underlying tariff is created, then delegates to tariff
func (p *cachingProxy) Rates() (api.Rates, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.tariff == nil {
		if res, err := p.cacheGet(for24hrs()); err == nil {
			return slices.Clone(res.Rates), nil
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
func (p *cachingProxy) Type() api.TariffType {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.tariff == nil {
		if res, err := p.cacheGet(for24hrs()); err == nil {
			return res.Type
		}

		p.createInstance()
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

func (p *cachingProxy) cacheGet(until time.Time) (*cached, error) {
	if p.cached == nil {
		res, err := cacheGet(p.key)
		if err != nil {
			return nil, err
		}

		p.cached = res
	}

	if !ratesValid(p.cached.Rates, until) {
		return nil, errors.New("not enough rates")
	}

	return p.cached, nil
}

func (p *cachingProxy) cachePut(typ api.TariffType, rates api.Rates) error {
	hash := sha256.Sum256(fmt.Append(nil, rates))
	if hash == p.hash {
		return nil
	}

	p.hash = hash
	return cachePut(p.key, typ, rates)
}

func for24hrs() time.Time {
	return time.Now().Add(24 * time.Hour)
}

func untilEndOfTomorrow() time.Time {
	return now.BeginningOfDay().AddDate(0, 0, 2)
}

func ratesValid(rr api.Rates, until time.Time) bool {
	return len(rr) > 0 && !rr[len(rr)-1].End.Before(until)
}
