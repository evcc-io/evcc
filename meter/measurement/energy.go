package measurement

import (
	"context"
	"fmt"

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

	return powerG, energyG, returnG, nil
}
