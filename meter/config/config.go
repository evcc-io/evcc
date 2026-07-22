package config

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	reg "github.com/evcc-io/evcc/util/registry"
	"github.com/spf13/cast"
)

var Registry = reg.New[api.Meter]("meter")

// efficiencyParam is the battery efficiency setting. It is handled here instead of
// in the device implementations since it is not used by the device itself.
const efficiencyParam = "efficiency"

// NewFromConfig creates meter from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]any) (api.Meter, error) {
	factory, err := Registry.Get(strings.ToLower(typ))
	if err != nil {
		return nil, err
	}

	// templates are re-entered with the rendered config, hence handled there
	var efficiency int64
	if !strings.EqualFold(typ, "template") {
		if efficiency, other, err = extractEfficiency(other); err != nil {
			return nil, fmt.Errorf("cannot create meter type '%s': %w", typ, err)
		}
	}

	v, err := factory(ctx, other)
	if err != nil {
		return nil, fmt.Errorf("cannot create meter type '%s': %w", util.TypeWithTemplateName(typ, other), err)
	}

	if efficiency > 0 {
		caps, ok := v.(implement.Caps)
		if !ok {
			return nil, fmt.Errorf("cannot create meter type '%s': %s not supported", typ, efficiencyParam)
		}
		implement.Has(caps, implement.BatteryEfficiency(func() int64 { return efficiency }))
	}

	return v, nil
}

// extractEfficiency removes the efficiency setting from the config and returns its
// value in %. It returns 0 if the setting is not configured.
func extractEfficiency(other map[string]any) (int64, map[string]any, error) {
	val, ok := other[efficiencyParam]
	if !ok {
		return 0, other, nil
	}

	res := maps.Clone(other)
	delete(res, efficiencyParam)

	if val == nil || val == "" {
		return 0, res, nil
	}

	efficiency, err := cast.ToInt64E(val)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: %w", efficiencyParam, err)
	}

	if efficiency <= 0 || efficiency > 100 {
		return 0, nil, fmt.Errorf("%s: invalid value %v", efficiencyParam, val)
	}

	return efficiency, res, nil
}
