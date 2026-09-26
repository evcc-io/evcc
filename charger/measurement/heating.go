package measurement

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/plugin"
)

type Temperature struct {
	Temp             *plugin.Config // optional
	LimitTemp        *plugin.Config // optional
	TempHeating      *plugin.Config `mapstructure:"temp_heating"`       // optional, e.g. underfloor heating flow temperature
	LimitTempHeating *plugin.Config `mapstructure:"limittemp_heating"`  // optional
	TempWater        *plugin.Config `mapstructure:"temp_hotwater"`      // optional, e.g. hot water tank temperature
	LimitTempWater   *plugin.Config `mapstructure:"limittemp_hotwater"` // optional
}

func (cc *Temperature) Configure(ctx context.Context) (
	func() (float64, error),
	func() (int64, error),
	error,
) {
	tempG, err := cc.Temp.FloatGetter(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("temp: %w", err)
	}

	limitTempG, err := cc.LimitTemp.IntGetter(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("limit temp: %w", err)
	}

	return tempG, limitTempG, nil
}

// ConfigureHeating configures the optional heating circuit temperature and its limit
func (cc *Temperature) ConfigureHeating(ctx context.Context) (
	func() (float64, error),
	func() (int64, error),
	error,
) {
	tempHeatingG, err := cc.TempHeating.FloatGetter(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("temp heating: %w", err)
	}

	limitTempHeatingG, err := cc.LimitTempHeating.IntGetter(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("limit temp heating: %w", err)
	}

	return tempHeatingG, limitTempHeatingG, nil
}

// ConfigureWater configures the optional hot water temperature and its limit
func (cc *Temperature) ConfigureWater(ctx context.Context) (
	func() (float64, error),
	func() (int64, error),
	error,
) {
	tempWaterG, err := cc.TempWater.FloatGetter(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("temp water: %w", err)
	}

	limitTempWaterG, err := cc.LimitTempWater.IntGetter(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("limit temp water: %w", err)
	}

	return tempWaterG, limitTempWaterG, nil
}
