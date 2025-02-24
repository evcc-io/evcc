package measurement

import (
	"context"
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/plugin"
)

type Phases struct {
	Currents, Voltages, Powers []plugin.Config // optional
}

func (cc *Phases) Configure(ctx context.Context) (
	func() (float64, float64, float64, error),
	func() (float64, float64, float64, error),
	func() (float64, float64, float64, error),
	error,
) {
	currentsG, err := buildPhaseProviders(ctx, cc.Currents)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("currents: %w", err)
	}

	voltagesG, err := buildPhaseProviders(ctx, cc.Voltages)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("voltages: %w", err)
	}

	powersG, err := buildPhaseProviders(ctx, cc.Powers)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("powers: %w", err)
	}

	return currentsG, voltagesG, powersG, nil
}

// buildPhaseProviders returns phases getter for given config
func buildPhaseProviders(ctx context.Context, providers []plugin.Config) (func() (float64, float64, float64, error), error) {
	if len(providers) == 0 {
		return nil, nil
	}

	if len(providers) != 3 {
		return nil, errors.New("need one per phase, total three")
	}

	var phases [3]func() (float64, error)
	for idx, prov := range providers {
		c, err := prov.FloatGetter(ctx)
		if err != nil {
			return nil, fmt.Errorf("[%d] %w", idx, err)
		}

		phases[idx] = c
	}

	return CombinePhases(phases), nil
}

// CombinePhases combines phase getters into combined api function
func CombinePhases(g [3]func() (float64, error)) func() (float64, float64, float64, error) {
	return func() (float64, float64, float64, error) {
		var res [3]float64
		for idx, currentG := range g {
			c, err := currentG()
			if err != nil {
				return 0, 0, 0, err
			}

			res[idx] = c
		}

		return res[0], res[1], res[2], nil
	}
}
