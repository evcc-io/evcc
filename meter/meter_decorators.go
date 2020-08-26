package meter

// This file has been generated - do not modify

import (
	"github.com/andig/evcc/api"
)

func decorateMeter(base api.Meter, meterEnergy func() (float64, error), meterCurrent func() (float64, float64, float64, error)) api.Meter {
	switch {
	case meterCurrent == nil && meterEnergy == nil:
		return base

	case meterCurrent == nil && meterEnergy != nil:
		return &struct{
			api.Meter
			api.MeterEnergy
		}{
			Meter: base,
			MeterEnergy: &decorateMeterMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meterCurrent != nil && meterEnergy == nil:
		return &struct{
			api.Meter
			api.MeterCurrent
		}{
			Meter: base,
			MeterCurrent: &decorateMeterMeterCurrentImpl{
				meterCurrent: meterCurrent,
			},
		}

	case meterCurrent != nil && meterEnergy != nil:
		return &struct{
			api.Meter
			api.MeterCurrent
			api.MeterEnergy
		}{
			Meter: base,
			MeterCurrent: &decorateMeterMeterCurrentImpl{
				meterCurrent: meterCurrent,
			},
			MeterEnergy: &decorateMeterMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}
	}

	return nil
}

type decorateMeterMeterCurrentImpl struct {
	meterCurrent func() (float64, float64, float64, error)
}

func (impl *decorateMeterMeterCurrentImpl) Currents() (float64, float64, float64, error) {
	return impl.meterCurrent()
}

type decorateMeterMeterEnergyImpl struct {
	meterEnergy func() (float64, error)
}

func (impl *decorateMeterMeterEnergyImpl) TotalEnergy() (float64, error) {
	return impl.meterEnergy()
}
