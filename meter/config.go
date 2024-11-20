package meter

import (
	"context"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	reg "github.com/evcc-io/evcc/util/registry"
)

var registry = reg.New[api.Meter]("meter")

// Types returns the list of types
func Types() []string {
	return registry.Types()
}

// NewFromConfig creates meter from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]interface{}) (api.Meter, error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err != nil {
		return nil, err
	}

	v, err := factory(ctx, other)
	if err != nil {
		err = fmt.Errorf("cannot create meter type '%s': %w", typ, err)
	}

	return v, err
}
