package tariff

import (
	"context"
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// CachingProxy wraps a tariff with caching
type CachingProxy struct {
	log *util.Logger
	mu  sync.Mutex

	ctx    context.Context
	typ    string
	config map[string]any

	tariff api.Tariff // Will be nil initially
}

var _ api.Tariff = (*CachingProxy)(nil)

// NewCachingProxy creates a proxy that controls tariff instantiation and caching
func NewCachingProxy(ctx context.Context, typ string, other map[string]any) (api.Tariff, error) {
	actualTyp := typ
	if typ == "template" {
		if template, ok := other["template"].(string); ok {
			actualTyp = template
		}
	}

	// cache, hash := NewSolarCacheManager(actualTyp, other)

	log := util.NewLogger(fmt.Sprintf("%s-%s", actualTyp, "hash"))

	proxy := &CachingProxy{
		log:    log,
		ctx:    ctx,
		typ:    typ,
		config: other,
	}

	if false { // TODO no data
		tariff, err := NewFromConfig(ctx, typ, other)
		if err != nil {
			return nil, err
		}

		proxy.tariff = tariff
	}

	return proxy, nil
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

	if p.tariff != nil {
		return p.tariff.Rates()
	}

	if true { // TODO get from cache
		return api.Rates{}, nil
	}

	p.createInstance()

	return p.tariff.Rates()
}

// Type returns the tariff type
func (p *CachingProxy) Type() api.TariffType {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.tariff != nil {
		return p.tariff.Type()
	}

	if true { // TODO get from cache
		return 0
	}

	p.createInstance()

	return p.tariff.Type()
}
