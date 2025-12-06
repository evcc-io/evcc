package hems

import (
	"context"
	"errors"
	"strings"

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
	switch strings.ToLower(typ) {
	case "sma", "shm", "semp":
		return nil, errors.New("breaking change: Sunny Home Manager integration is always on. See https://github.com/evcc-io/evcc/releases and https://docs.evcc.io/en/docs/integrations/sma-sunny-home-manager")
	default:
		return config.NewFromConfig(ctx, typ, other, site)
	}
}
