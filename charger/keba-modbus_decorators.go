package charger

// Code generated by github.com/evcc-io/evcc/cmd/tools/decorate.go. DO NOT EDIT.

import (
	"github.com/evcc-io/evcc/api"
)

func decorateKeba(base *Keba, meter func() (float64, error), meterEnergy func() (float64, error), phaseCurrents func() (float64, float64, float64, error), phaseVoltages func() (float64, float64, float64, error), phaseSwitcher func(int) error) api.Charger {
	switch {
	case meter == nil && meterEnergy == nil && phaseCurrents == nil && phaseSwitcher == nil && phaseVoltages == nil:
		return base

	case meter != nil && meterEnergy == nil && phaseCurrents == nil && phaseSwitcher == nil && phaseVoltages == nil:
		return &struct {
			*Keba
			api.Meter
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
		}

	case meter == nil && meterEnergy != nil && phaseCurrents == nil && phaseSwitcher == nil && phaseVoltages == nil:
		return &struct {
			*Keba
			api.MeterEnergy
		}{
			Keba: base,
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meter != nil && meterEnergy != nil && phaseCurrents == nil && phaseSwitcher == nil && phaseVoltages == nil:
		return &struct {
			*Keba
			api.Meter
			api.MeterEnergy
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meter == nil && meterEnergy == nil && phaseCurrents != nil && phaseSwitcher == nil && phaseVoltages == nil:
		return &struct {
			*Keba
			api.PhaseCurrents
		}{
			Keba: base,
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
		}

	case meter != nil && meterEnergy == nil && phaseCurrents != nil && phaseSwitcher == nil && phaseVoltages == nil:
		return &struct {
			*Keba
			api.Meter
			api.PhaseCurrents
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
		}

	case meter == nil && meterEnergy != nil && phaseCurrents != nil && phaseSwitcher == nil && phaseVoltages == nil:
		return &struct {
			*Keba
			api.MeterEnergy
			api.PhaseCurrents
		}{
			Keba: base,
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
		}

	case meter != nil && meterEnergy != nil && phaseCurrents != nil && phaseSwitcher == nil && phaseVoltages == nil:
		return &struct {
			*Keba
			api.Meter
			api.MeterEnergy
			api.PhaseCurrents
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
		}

	case meter == nil && meterEnergy == nil && phaseCurrents == nil && phaseSwitcher == nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.PhaseVoltages
		}{
			Keba: base,
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case meter != nil && meterEnergy == nil && phaseCurrents == nil && phaseSwitcher == nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.Meter
			api.PhaseVoltages
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case meter == nil && meterEnergy != nil && phaseCurrents == nil && phaseSwitcher == nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.MeterEnergy
			api.PhaseVoltages
		}{
			Keba: base,
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case meter != nil && meterEnergy != nil && phaseCurrents == nil && phaseSwitcher == nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.Meter
			api.MeterEnergy
			api.PhaseVoltages
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case meter == nil && meterEnergy == nil && phaseCurrents != nil && phaseSwitcher == nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.PhaseCurrents
			api.PhaseVoltages
		}{
			Keba: base,
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case meter != nil && meterEnergy == nil && phaseCurrents != nil && phaseSwitcher == nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.Meter
			api.PhaseCurrents
			api.PhaseVoltages
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case meter == nil && meterEnergy != nil && phaseCurrents != nil && phaseSwitcher == nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.MeterEnergy
			api.PhaseCurrents
			api.PhaseVoltages
		}{
			Keba: base,
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case meter != nil && meterEnergy != nil && phaseCurrents != nil && phaseSwitcher == nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.Meter
			api.MeterEnergy
			api.PhaseCurrents
			api.PhaseVoltages
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case meter == nil && meterEnergy == nil && phaseCurrents == nil && phaseSwitcher != nil && phaseVoltages == nil:
		return &struct {
			*Keba
			api.PhaseSwitcher
		}{
			Keba: base,
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
		}

	case meter != nil && meterEnergy == nil && phaseCurrents == nil && phaseSwitcher != nil && phaseVoltages == nil:
		return &struct {
			*Keba
			api.Meter
			api.PhaseSwitcher
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
		}

	case meter == nil && meterEnergy != nil && phaseCurrents == nil && phaseSwitcher != nil && phaseVoltages == nil:
		return &struct {
			*Keba
			api.MeterEnergy
			api.PhaseSwitcher
		}{
			Keba: base,
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
		}

	case meter != nil && meterEnergy != nil && phaseCurrents == nil && phaseSwitcher != nil && phaseVoltages == nil:
		return &struct {
			*Keba
			api.Meter
			api.MeterEnergy
			api.PhaseSwitcher
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
		}

	case meter == nil && meterEnergy == nil && phaseCurrents != nil && phaseSwitcher != nil && phaseVoltages == nil:
		return &struct {
			*Keba
			api.PhaseCurrents
			api.PhaseSwitcher
		}{
			Keba: base,
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
		}

	case meter != nil && meterEnergy == nil && phaseCurrents != nil && phaseSwitcher != nil && phaseVoltages == nil:
		return &struct {
			*Keba
			api.Meter
			api.PhaseCurrents
			api.PhaseSwitcher
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
		}

	case meter == nil && meterEnergy != nil && phaseCurrents != nil && phaseSwitcher != nil && phaseVoltages == nil:
		return &struct {
			*Keba
			api.MeterEnergy
			api.PhaseCurrents
			api.PhaseSwitcher
		}{
			Keba: base,
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
		}

	case meter != nil && meterEnergy != nil && phaseCurrents != nil && phaseSwitcher != nil && phaseVoltages == nil:
		return &struct {
			*Keba
			api.Meter
			api.MeterEnergy
			api.PhaseCurrents
			api.PhaseSwitcher
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
		}

	case meter == nil && meterEnergy == nil && phaseCurrents == nil && phaseSwitcher != nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.PhaseSwitcher
			api.PhaseVoltages
		}{
			Keba: base,
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case meter != nil && meterEnergy == nil && phaseCurrents == nil && phaseSwitcher != nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.Meter
			api.PhaseSwitcher
			api.PhaseVoltages
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case meter == nil && meterEnergy != nil && phaseCurrents == nil && phaseSwitcher != nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.MeterEnergy
			api.PhaseSwitcher
			api.PhaseVoltages
		}{
			Keba: base,
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case meter != nil && meterEnergy != nil && phaseCurrents == nil && phaseSwitcher != nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.Meter
			api.MeterEnergy
			api.PhaseSwitcher
			api.PhaseVoltages
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case meter == nil && meterEnergy == nil && phaseCurrents != nil && phaseSwitcher != nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.PhaseCurrents
			api.PhaseSwitcher
			api.PhaseVoltages
		}{
			Keba: base,
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case meter != nil && meterEnergy == nil && phaseCurrents != nil && phaseSwitcher != nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.Meter
			api.PhaseCurrents
			api.PhaseSwitcher
			api.PhaseVoltages
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case meter == nil && meterEnergy != nil && phaseCurrents != nil && phaseSwitcher != nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.MeterEnergy
			api.PhaseCurrents
			api.PhaseSwitcher
			api.PhaseVoltages
		}{
			Keba: base,
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}

	case meter != nil && meterEnergy != nil && phaseCurrents != nil && phaseSwitcher != nil && phaseVoltages != nil:
		return &struct {
			*Keba
			api.Meter
			api.MeterEnergy
			api.PhaseCurrents
			api.PhaseSwitcher
			api.PhaseVoltages
		}{
			Keba: base,
			Meter: &decorateKebaMeterImpl{
				meter: meter,
			},
			MeterEnergy: &decorateKebaMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhaseCurrents: &decorateKebaPhaseCurrentsImpl{
				phaseCurrents: phaseCurrents,
			},
			PhaseSwitcher: &decorateKebaPhaseSwitcherImpl{
				phaseSwitcher: phaseSwitcher,
			},
			PhaseVoltages: &decorateKebaPhaseVoltagesImpl{
				phaseVoltages: phaseVoltages,
			},
		}
	}

	return nil
}

type decorateKebaMeterImpl struct {
	meter func() (float64, error)
}

func (impl *decorateKebaMeterImpl) CurrentPower() (float64, error) {
	return impl.meter()
}

type decorateKebaMeterEnergyImpl struct {
	meterEnergy func() (float64, error)
}

func (impl *decorateKebaMeterEnergyImpl) TotalEnergy() (float64, error) {
	return impl.meterEnergy()
}

type decorateKebaPhaseCurrentsImpl struct {
	phaseCurrents func() (float64, float64, float64, error)
}

func (impl *decorateKebaPhaseCurrentsImpl) Currents() (float64, float64, float64, error) {
	return impl.phaseCurrents()
}

type decorateKebaPhaseSwitcherImpl struct {
	phaseSwitcher func(int) error
}

func (impl *decorateKebaPhaseSwitcherImpl) Phases1p3p(phases int) error {
	return impl.phaseSwitcher(phases)
}

type decorateKebaPhaseVoltagesImpl struct {
	phaseVoltages func() (float64, float64, float64, error)
}

func (impl *decorateKebaPhaseVoltagesImpl) Voltages() (float64, float64, float64, error) {
	return impl.phaseVoltages()
}
