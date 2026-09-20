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

	tariff   api.Tariff
	err      error
	retryAt  time.Time // earliest next creation attempt
	creating bool      // creation in progress
}

var _ api.Tariff = (*Wrapper)(nil)

// NewWrapper creates an offline tariff wrapper
func NewWrapper(ctx context.Context, typ string, other map[string]any, err error) api.Tariff {
	v := &Wrapper{
		log:     util.NewLogger("tariff"),
		ctx:     ctx,
		typ:     typ,
		config:  other,
		err:     err,
		retryAt: time.Now().Add(retryInterval),
	}

	return v
}

// WrappedConfig indicates a device with wrapped configuration
func (v *Wrapper) WrappedConfig() (string, map[string]any) {
	return v.typ, v.config
}

// instance returns the tariff once created. Creation is retried in the background at most once per retryInterval.
func (v *Wrapper) instance() api.Tariff {
	if v.tariff != nil || v.creating || time.Now().Before(v.retryAt) {
		return v.tariff
	}

	v.creating = true

	go func() {
		t, err := NewFromConfig(v.ctx, v.typ, v.config)

		v.mu.Lock()
		defer v.mu.Unlock()

		v.creating = false

		if err != nil {
			v.err = err
			v.retryAt = time.Now().Add(retryInterval)
			v.log.WARN.Printf("creating tariff failed: %v", err)
			return
		}

		v.tariff = t
		v.log.INFO.Printf("tariff available: %s", util.TypeWithTemplateName(v.typ, v.config))
	}()

	return nil
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
