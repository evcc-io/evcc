package circuit

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/api"
)

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates circuit from custom yaml config
func NewConfigurableFromConfig(ctx context.Context, other map[string]any) (api.Circuit, error) {
	typ, ok := other["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing type in custom circuit config")
	}
	delete(other, "type")
	return NewFromConfig(ctx, typ, other)
}
