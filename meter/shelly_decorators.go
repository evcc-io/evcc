package meter

// Code generated by github.com/evcc-io/evcc/cmd/tools/decorate.go. DO NOT EDIT.

import (
	"github.com/evcc-io/evcc/api"
)

func decorateShelly(base *Shelly, phaseVoltages func() (float64, float64, float64, error), phaseCurrents func() (float64, float64, float64, error)) api.Meter {
	switch {
	case phaseCurrents == nil && phaseVoltages == nil:
		return base

	case phaseCurrents == nil && phaseVoltages != nil:
		return &struct {
			*Shelly
			api.PhaseVoltages
		}{
			Shelly: base,
			PhaseVoltages: &decorateShellyPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case phaseCurrents != nil && phaseVoltages == nil:
		return &struct {
			*Shelly
			api.PhaseCurrents
		}{
			Shelly: base,
			PhaseCurrents: &decorateShellyPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
		}

	case phaseCurrents != nil && phaseVoltages != nil:
		return &struct {
			*Shelly
			api.PhaseCurrents
			api.PhaseVoltages
		}{
			Shelly: base,
			PhaseCurrents: &decorateShellyPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
			PhaseVoltages: &decorateShellyPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}
	}

	return nil
}

type decorateShellyPhaseCurrentsImpl struct {
	phaseCurrents func() (float64, float64, float64, error)
}

func (impl *decorateShellyPhaseCurrentsImpl) Currents() (float64, float64, float64, error) {
	return impl.phaseCurrents()
}

type decorateShellyPhaseVoltagesImpl struct {
	phaseVoltages func() (float64, float64, float64, error)
}

func (impl *decorateShellyPhaseVoltagesImpl) Voltages() (float64, float64, float64, error) {
	return impl.phaseVoltages()
}
