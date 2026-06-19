package measurement

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin"
)

type Energy struct {
	Power        plugin.Config
	Energy       *plugin.Config // optional
	ReturnEnergy *plugin.Config // optional
}

func (cc *Energy) Configure(ctx context.Context) (
	func() (float64, error),
	func() (float64, error),
	func() (float64, error),
	error,
) {
	powerG, err := cc.Power.FloatGetter(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("power: %w", err)
	}

	energyG, err := cc.Energy.FloatGetter(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("energy: %w", err)
	}

	returnG, err := cc.ReturnEnergy.FloatGetter(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("returnEnergy: %w", err)
	}

	return powerG, nonZeroEnergy(energyG), nonZeroEnergy(returnG), nil
}

// nonZeroEnergy reports a zero lifetime energy reading as api.ErrNotAvailable.
// Some inverters reset their total counter to 0 at night; this keeps the last value (#30951).
func nonZeroEnergy(g func() (float64, error)) func() (float64, error) {
	if g == nil {
		return nil
	}

	return func() (float64, error) {
		f, err := g()
		if err == nil && f == 0 {
			return 0, api.ErrNotAvailable
		}
		return f, err
	}
}
