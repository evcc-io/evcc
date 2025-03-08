package charger

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/config"
)

var registry = config.Registry

// Register ist the public api for registering a type
func Register(typ string, newFun func(context.Context, map[string]any) (api.Charger, error)) {
	registry.AddCtx(typ, newFun)
}

// Types returns the list of types
func Types() []string {
	return registry.Types()
}

// NewFromConfig creates charger from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]any) (api.Charger, error) {
	return config.NewFromConfig(ctx, typ, other)
}
