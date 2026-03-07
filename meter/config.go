package meter

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/config"
	"github.com/spf13/cast"
)

var registry = config.Registry

// Types returns the list of types
func Types() []string {
	return registry.Types()
}

// NewFromConfig creates meter from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]any) (api.Meter, error) {
	reversed := cast.ToBool(other["reversed"])
	if reversed {
		other = withoutReverse(other)
	}

	meter, err := config.NewFromConfig(ctx, typ, other)
	if err != nil {
		return nil, err
	}

	if reversed {
		meter = Reverse(meter)
	}

	return meter, nil
}

func withoutReverse(other map[string]any) map[string]any {
	if len(other) == 0 {
		return other
	}

	cloned := make(map[string]any, len(other))
	for k, v := range other {
		if k == "reversed" {
			continue
		}
		cloned[k] = v
	}

	return cloned
}
