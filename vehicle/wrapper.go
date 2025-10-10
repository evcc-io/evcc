package vehicle

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Wrapper wraps an api.Vehicle to capture initialization errors
type Wrapper struct {
	*embed
	typ    string
	config map[string]any
	err    error
}

// NewWrapper creates an offline Vehicle wrapper
func NewWrapper(ctx context.Context, name, typ string, other map[string]any, err error) api.Vehicle {
	var cc struct {
		embed `mapstructure:",squash"`
		Other map[string]any `mapstructure:",remain"`
	}

	// try to decode vehicle-specific config and look for embedded attributes
	_ = util.DecodeOther(other, &cc)

	v := &Wrapper{
		embed:  cc.embed.withContext(ctx),
		typ:    typ,
		config: cc.Other,
		err:    fmt.Errorf("vehicle not available: %w", err),
	}

	v.Features_ = append(v.Features_, api.Offline, api.Retryable)

	return v
}

// WrappedConfig indicates a device with wrapped configuration
func (v *Wrapper) WrappedConfig() (string, map[string]any) {
	return v.typ, v.config
}

var _ api.Battery = (*Wrapper)(nil)

// Soc implements the api.Battery interface
func (v *Wrapper) Soc() (float64, error) {
	return 0, v.err
}
