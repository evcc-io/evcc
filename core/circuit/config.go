package circuit

import (
	"context"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	reg "github.com/evcc-io/evcc/util/registry"
)

var registry = reg.New[api.Circuit]("circuit")

// NewFromConfig creates api.Circuit from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]any) (api.Circuit, error) {
	// treat any non-template circuit as custom in order for registry lookup to work
	if typ == "" {
		typ = api.Custom
	}

	factory, err := registry.Get(strings.ToLower(typ))
	if err != nil {
		return nil, err
	}

	v, err := factory(ctx, other)
	if err != nil {
		err = fmt.Errorf("cannot create circuit type '%s': %w", util.TypeWithTemplateName(typ, other), err)
	}

	return v, err
}

func Root() api.Circuit {
	for _, dev := range config.Circuits().Devices() {
		if c := dev.Instance(); c.GetParent() == nil {
			return c
		}
	}
	return nil
}
