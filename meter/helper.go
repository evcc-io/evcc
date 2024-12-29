package meter

import (
	"context"
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/provider"
)

// BuildMeasurements returns typical meter measurement getters from config
func BuildMeasurements(ctx context.Context, power, energyImport, export *provider.Config) (func() (float64, error), func() (float64, error), func() (float64, error), error) {
	var powerG func() (float64, error)
	if power != nil {
		var err error
		powerG, err = provider.NewFloatGetterFromConfig(ctx, *power)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("power: %w", err)
		}
	}

	var importG func() (float64, error)
	if energyImport != nil {
		var err error
		importG, err = provider.NewFloatGetterFromConfig(ctx, *energyImport)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("import: %w", err)
		}
	}

	var exportG func() (float64, error)
	if export != nil {
		var err error
		exportG, err = provider.NewFloatGetterFromConfig(ctx, *export)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("export: %w", err)
		}
	}

	return powerG, importG, exportG, nil
}

// BuildPhaseMeasurements returns typical meter measurement getters from config
func BuildPhaseMeasurements(ctx context.Context, currents, voltages, powers []provider.Config) (
	func() (float64, float64, float64, error),
	func() (float64, float64, float64, error),
	func() (float64, float64, float64, error),
	error,
) {
	currentsG, err := buildPhaseProviders(ctx, currents)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("currents: %w", err)
	}

	voltagesG, err := buildPhaseProviders(ctx, voltages)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("voltages: %w", err)
	}

	powersG, err := buildPhaseProviders(ctx, powers)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("powers: %w", err)
	}

	return currentsG, voltagesG, powersG, nil
}

// buildPhaseProviders returns phases getter for given config
func buildPhaseProviders(ctx context.Context, providers []provider.Config) (func() (float64, float64, float64, error), error) {
	if len(providers) == 0 {
		return nil, nil
	}

	if len(providers) != 3 {
		return nil, errors.New("need one per phase, total three")
	}

	var phases [3]func() (float64, error)
	for idx, prov := range providers {
		c, err := provider.NewFloatGetterFromConfig(ctx, prov)
		if err != nil {
			return nil, fmt.Errorf("[%d] %w", idx, err)
		}

		phases[idx] = c
	}

	return collectPhaseProviders(phases), nil
}

// collectPhaseProviders combines phase getters into combined api function
func collectPhaseProviders(g [3]func() (float64, error)) func() (float64, float64, float64, error) {
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
