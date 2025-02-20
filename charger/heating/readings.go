package heating

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/plugin"
)

type Readings struct {
	Power     *plugin.Config // optional
	Energy    *plugin.Config // optional
	Temp      *plugin.Config // optional
	LimitTemp *plugin.Config // optional
}

func (cc *Readings) Configure(ctx context.Context) (
	func() (float64, error),
	func() (float64, error),
	func() (float64, error),
	func() (int64, error),
	error,
) {
	// decorate power
	powerG, err := cc.Power.FloatGetter(ctx)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("power: %w", err)
	}

	// decorate energy
	energyG, err := cc.Energy.FloatGetter(ctx)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("energy: %w", err)
	}

	// decorate temp
	tempG, err := cc.Temp.FloatGetter(ctx)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("temp: %w", err)
	}

	limitTempG, err := cc.LimitTemp.IntGetter(ctx)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("limit temp: %w", err)
	}

	return powerG, energyG, tempG, limitTempG, nil
}
