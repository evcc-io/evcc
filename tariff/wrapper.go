package tariff

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// retryInterval is the minimum time between attempts to create an unavailable tariff
const retryInterval = 15 * time.Minute

// Wrapper wraps an api.Tariff that could not be created and retries creating it
type Wrapper struct {
	mu     sync.Mutex
	log    *util.Logger
	ctx    context.Context
	typ    string
	config map[string]any

	tariff    api.Tariff
	err       error
	retriedAt time.Time // last creation attempt
}

var _ api.Tariff = (*Wrapper)(nil)

// NewWrapper creates an offline tariff wrapper
func NewWrapper(ctx context.Context, typ string, other map[string]any, err error) api.Tariff {
	return &Wrapper{
		log:    util.NewLogger("tariff"),
		ctx:    ctx,
		typ:    typ,
		config: other,
		err:    err,
	}
}

// WrappedConfig indicates a device with wrapped configuration
func (v *Wrapper) WrappedConfig() (string, map[string]any) {
	return v.typ, v.config
}

// instance returns the tariff once created. Creation is retried at most once per retryInterval.
func (v *Wrapper) instance() api.Tariff {
	if v.tariff != nil || time.Since(v.retriedAt) < retryInterval {
		return v.tariff
	}

	v.retriedAt = time.Now()

	t, err := NewFromConfig(v.ctx, v.typ, v.config)
	if err != nil {
		v.err = err
		v.log.WARN.Printf("creating tariff failed: %v", err)
		return nil
	}

	v.tariff = t
	v.log.INFO.Printf("tariff available: %s", util.TypeWithTemplateName(v.typ, v.config))

	return t
}

// Rates implements the api.Tariff interface
func (v *Wrapper) Rates() (api.Rates, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t := v.instance(); t != nil {
		return t.Rates()
	}

	return nil, fmt.Errorf("tariff not available: %w", v.err)
}

// Type implements the api.Tariff interface
func (v *Wrapper) Type() api.TariffType {
	v.mu.Lock()
	defer v.mu.Unlock()

	if t := v.instance(); t != nil {
		return t.Type()
	}

	return 0
}
