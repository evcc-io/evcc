package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/hems"
	reg "github.com/evcc-io/evcc/util/registry"
)

var Registry = reg.New[hems.API]("hems")

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
