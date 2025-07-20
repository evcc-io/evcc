package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	reg "github.com/evcc-io/evcc/util/registry"
)

var Registry = reg.New[api.Charger]("charger")

// NewFromConfig creates charger from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]interface{}) (api.Charger, error) {
	factory, err := Registry.Get(strings.ToLower(typ))
	if err != nil {
		return nil, err
	}

	v, err := factory(ctx, other)
	if err != nil {
		err = fmt.Errorf("cannot create charger type '%s': %w", typ, err)
	}

	return v, err
}
