package tariff

import (
	"context"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	reg "github.com/evcc-io/evcc/util/registry"
)

var registry = reg.New[api.Tariff]("tariff")

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
		return nil, fmt.Errorf("cannot create tariff type '%s': %w", typ, err)
	}

	// check slot length
	if rr, err := v.Rates(); err == nil && len(rr) > 0 && rr[0].End.Sub(rr[0].Start) == SlotDuration {
		return v, nil
	}

	return &SlotWrapper{Tariff: v}, nil
}
