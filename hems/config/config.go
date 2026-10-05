package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/hems"
	reg "github.com/evcc-io/evcc/hems/registry"
)

var Registry = reg.New[hems.API]("hems")

// AddCtx registers a factory returning a concrete type that implements hems.API
func AddCtx[T hems.API](name string, factory func(context.Context, map[string]any, site.API) (T, error)) {
	Registry.AddCtx(name, func(ctx context.Context, other map[string]any, site site.API) (hems.API, error) {
		return factory(ctx, other, site)
	})
}

// NewFromConfig creates hems from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]any, site site.API) (hems.API, error) {
	factory, err := Registry.Get(strings.ToLower(typ))
	if err != nil {
		return nil, err
	}

	v, err := factory(ctx, other, site)
	if err != nil {
		return nil, fmt.Errorf("cannot create hems type '%s': %w", typ, err)
	}

	return v, nil
}
