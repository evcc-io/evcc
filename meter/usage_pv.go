package meter

import (
	"context"

	"github.com/evcc-io/evcc/util"
)

type pvMaxACPower struct {
	MaxACPower float64
}

// var _ api.MaxACPowerGetter = (*pvMaxACPower)(nil)

// Decorator returns the max AC power decorator
func (m *pvMaxACPower) Decorator() func() float64 {
	return staticFloat(m.MaxACPower)
}

type pvMaxACPowerCtx struct {
	MaxACPower any // static W value or float plugin
}

// var _ api.MaxACPowerGetter = (*pvMaxACPowerCtx)(nil)

// Decorator returns the max AC power decorator. MaxACPower may be a static value or a
// float plugin like the SunSpec nameplate rating. Since the rating is constant, the
// plugin is read once. Unavailable or zero means not configured.
func (m *pvMaxACPowerCtx) Decorator(ctx context.Context) func() float64 {
	get, err := resolveFloat(ctx, m.MaxACPower)
	if err != nil {
		// not all devices expose the rating- must not fail configuration
		util.NewLogger("meter").WARN.Printf("maxacpower: %v", err)
	}
	return staticFloat(floatOr0(get))
}
