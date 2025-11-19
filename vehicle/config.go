package vehicle

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	reg "github.com/evcc-io/evcc/util/registry"
)

const (
	expiry   = 5 * time.Minute  // maximum response age before refresh
	interval = 15 * time.Minute // refresh interval when charging
)

var registry = reg.New[api.Vehicle]("vehicle")

// Types returns the list of types
func Types() []string {
	return registry.Types()
}

// NewFromConfig creates vehicle from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]any) (api.Vehicle, error) {
	var cc struct {
		Cloud bool
		Other map[string]any `mapstructure:",remain"`
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Cloud {
		cc.Other["brand"] = typ
		typ = "cloud"
	}

	factory, err := registry.Get(strings.ToLower(typ))
	if err != nil {
		return nil, err
	}

	v, err := factory(ctx, cc.Other)
	if err != nil {
		return nil, fmt.Errorf("cannot create vehicle type '%s': %w", typ, err)
	}

	return v, nil
}
