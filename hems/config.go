package hems

import (
	"context"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/config"
	"github.com/evcc-io/evcc/hems/hems"
)

var registry = config.Registry

// Types returns the list of types
func Types() []string {
	return registry.Types()
}

// NewFromConfig creates hems from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]any, site site.API) (hems.API, error) {
	return config.NewFromConfig(ctx, typ, other)
}
