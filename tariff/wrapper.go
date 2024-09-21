package tariff

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
)

// Wrapper wraps an api.Tariff to capture initialization errors
type Wrapper struct {
	typ    string
	config map[string]interface{}
	err    error
}

// NewWrapper creates an offline tariff wrapper
func NewWrapper(typ string, other map[string]interface{}, err error) api.Tariff {
	v := &Wrapper{
		typ:    typ,
		config: other,
		err:    fmt.Errorf("tariff not available: %w", err),
	}

	return v
}

func (t *Wrapper) Rates() (api.Rates, error) {
	return nil, t.err
}

func (t *Wrapper) Type() api.TariffType {
	return 0
}
