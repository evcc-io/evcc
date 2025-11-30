package meter

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/config"
)

var registry = config.Registry

// Types returns the list of types
func Types() []string {
	return registry.Types()
}

// NewFromConfig creates meter from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]any) (api.Meter, error) {
	return config.NewFromConfig(ctx, typ, other)
}
