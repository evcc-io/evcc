package charger

// This file has been generated - do not modify

import (
	"github.com/andig/evcc/api"
)

func decorate(base api.Charger, meter func() (float64, error), meterEnergy func() (float64, error), meterCurrent func() (float64, float64, float64, error)) api.Charger {
	switch {
	case meter == nil && meterCurrent == nil && meterEnergy == nil:
		return base

	case meter != nil && meterCurrent == nil && meterEnergy == nil:
		return &struct{
			api.Charger
			api.Meter
		}{
			Charger: base,
			Meter: &decorateMeterImpl{
				meter: meter,
			},
		}

	case meter == nil && meterCurrent == nil && meterEnergy != nil:
		return &struct{
			api.Charger
			api.MeterEnergy
		}{
			Charger: base,
			MeterEnergy: &decorateMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meter != nil && meterCurrent == nil && meterEnergy != nil:
		return &struct{
			api.Charger
			api.Meter
			api.MeterEnergy
		}{
			Charger: base,
			Meter: &decorateMeterImpl{
				meter: meter,
			},
			MeterEnergy: &decorateMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meter == nil && meterCurrent != nil && meterEnergy == nil:
		return &struct{
			api.Charger
			api.MeterCurrent
		}{
			Charger: base,
			MeterCurrent: &decorateMeterCurrentImpl{
				meterCurrent: meterCurrent,
			},
		}

	case meter != nil && meterCurrent != nil && meterEnergy == nil:
		return &struct{
			api.Charger
			api.Meter
			api.MeterCurrent
		}{
			Charger: base,
			Meter: &decorateMeterImpl{
				meter: meter,
			},
			MeterCurrent: &decorateMeterCurrentImpl{
				meterCurrent: meterCurrent,
			},
		}

	case meter == nil && meterCurrent != nil && meterEnergy != nil:
		return &struct{
			api.Charger
			api.MeterCurrent
			api.MeterEnergy
		}{
			Charger: base,
			MeterCurrent: &decorateMeterCurrentImpl{
				meterCurrent: meterCurrent,
			},
			MeterEnergy: &decorateMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meter != nil && meterCurrent != nil && meterEnergy != nil:
		return &struct{
			api.Charger
			api.Meter
			api.MeterCurrent
			api.MeterEnergy
		}{
			Charger: base,
			Meter: &decorateMeterImpl{
				meter: meter,
			},
			MeterCurrent: &decorateMeterCurrentImpl{
				meterCurrent: meterCurrent,
			},
			MeterEnergy: &decorateMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}
	}

	return nil
}

type decorateMeterImpl struct {
	meter func() (float64, error)
}

func (impl *decorateMeterImpl) CurrentPower() (float64, error) {
	return impl.meter()
}

type decorateMeterCurrentImpl struct {
	meterCurrent func() (float64, float64, float64, error)
}

func (impl *decorateMeterCurrentImpl) Currents() (float64, float64, float64, error) {
	return impl.meterCurrent()
}

type decorateMeterEnergyImpl struct {
	meterEnergy func() (float64, error)
}

func (impl *decorateMeterEnergyImpl) TotalEnergy() (float64, error) {
	return impl.meterEnergy()
}
