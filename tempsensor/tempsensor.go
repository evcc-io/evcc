package tempsensor

import (
	"context"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	reg "github.com/evcc-io/evcc/util/registry"
)

var registry = reg.New[api.Battery]("tempsensor")

// NewFromConfig creates temp sensor from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]any) (api.Battery, error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err != nil {
		return nil, err
	}

	v, err := factory(ctx, other)
	if err != nil {
		err = fmt.Errorf("cannot create temp sensor type '%s': %w", util.TypeWithTemplateName(typ, other), err)
	}

	return v, err
}
