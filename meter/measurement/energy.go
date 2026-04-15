package measurement

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/plugin"
)

type Energy struct {
	Power  plugin.Config
	Energy *plugin.Config // TODO deprecated
	Import *plugin.Config // optional
}

func (cc *Energy) Configure(ctx context.Context) (
	func() (float64, error),
	func() (float64, error),
	error,
) {
	powerG, err := cc.Power.FloatGetter(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("power: %w", err)
	}

	// TODO deprecated - remove fallback
	if cc.Import == nil && cc.Energy != nil {
		cc.Import = cc.Energy
	}

	importG, err := cc.Import.FloatGetter(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("energy: %w", err)
	}

	return powerG, importG, nil
}
