package circuit

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/circuit"
)

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates api.Circuit from config
func NewConfigurableFromConfig(ctx context.Context, other map[string]any) (api.Circuit, error) {
	return circuit.NewFromConfig(ctx, other)
}
