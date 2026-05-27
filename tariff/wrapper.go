package tariff

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
)

// Wrapper wraps an api.Tariff to capture initialization errors
type Wrapper struct {
	typ    string
	config map[string]any
	err    error
}

// NewWrapper creates an offline tariff wrapper
func NewWrapper(typ string, other map[string]any, err error) api.Tariff {
	v := &Wrapper{
		typ:    typ,
		config: other,
		err:    fmt.Errorf("tariff not available: %w", err),
	}

	return v
}

// WrappedConfig indicates a device with wrapped configuration
func (v *Wrapper) WrappedConfig() (string, map[string]any) {
	return v.typ, v.config
}

// Rates implements the api.Tariff interface
func (t *Wrapper) Rates() (api.Rates, error) {
	return nil, t.err
}

// Type implements the api.Tariff interface
func (t *Wrapper) Type() api.TariffType {
	return 0
}
