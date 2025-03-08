package tariff

import (
	"context"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	reg "github.com/evcc-io/evcc/util/registry"
)

var registry = reg.New[api.Tariff]("tariff")

// Register ist the public api for registering a type
func Register(typ string, newFun func(context.Context, map[string]any) (api.Tariff, error)) {
	registry.AddCtx(typ, newFun)
}

// Types returns the list of types
func Types() []string {
	return registry.Types()
}

// NewFromConfig creates tariff from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]any) (api.Tariff, error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err != nil {
		return nil, err
	}

	v, err := factory(ctx, other)
	if err != nil {
		err = fmt.Errorf("cannot create tariff type '%s': %w", typ, err)
	}

	return v, err
}
