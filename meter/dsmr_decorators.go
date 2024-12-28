package meter

// Code generated by github.com/evcc-io/evcc/cmd/tools/decorate.go. DO NOT EDIT.

import (
	"github.com/evcc-io/evcc/api"
)

func decorateDsmr(base api.Meter, energyImport func() (float64, error), phaseCurrents func() (float64, float64, float64, error)) api.Meter {
	switch {
	case energyImport == nil && phaseCurrents == nil:
		return base

	case energyImport != nil && phaseCurrents == nil:
		return &struct {
			api.Meter
			api.EnergyImport
		}{
			Meter: base,
			EnergyImport: &decorateDsmrEnergyImportImpl{
				energyImport: energyImport,
			},
		}

	case energyImport == nil && phaseCurrents != nil:
		return &struct {
			api.Meter
			api.PhaseCurrents
		}{
			Meter: base,
			PhaseCurrents: &decorateDsmrPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
		}

	case energyImport != nil && phaseCurrents != nil:
		return &struct {
			api.Meter
			api.EnergyImport
			api.PhaseCurrents
		}{
			Meter: base,
			EnergyImport: &decorateDsmrEnergyImportImpl{
				energyImport: energyImport,
			},
			PhaseCurrents: &decorateDsmrPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
		}
	}

	return nil
}

type decorateDsmrEnergyImportImpl struct {
	energyImport func() (float64, error)
}

func (impl *decorateDsmrEnergyImportImpl) EnergyImport() (float64, error) {
	return impl.energyImport()
}

type decorateDsmrPhaseCurrentsImpl struct {
	phaseCurrents func() (float64, float64, float64, error)
}

func (impl *decorateDsmrPhaseCurrentsImpl) Currents() (float64, float64, float64, error) {
	return impl.phaseCurrents()
}
