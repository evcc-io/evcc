package meter

// This file has been generated - do not modify

import (
	"github.com/andig/evcc/api"
)

func meterDecorate(base api.Meter, meterEnergy func() (float64, error), meterCurrent func() (float64, float64, float64, error)) api.Meter {
	switch {
	case meterCurrent == nil && meterEnergy == nil:
		return base

	case meterCurrent == nil && meterEnergy != nil:
		return &struct{
			api.Meter
			api.MeterEnergy
		}{
			Meter: base,
			MeterEnergy: &meterDecorateMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meterCurrent != nil && meterEnergy == nil:
		return &struct{
			api.Meter
			api.MeterCurrent
		}{
			Meter: base,
			MeterCurrent: &meterDecorateMeterCurrentImpl{
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
			MeterCurrent: &meterDecorateMeterCurrentImpl{
				meterCurrent: meterCurrent,
			},
			MeterEnergy: &meterDecorateMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}
	}

	return nil
}

type meterDecorateMeterCurrentImpl struct {
	meterCurrent func() (float64, float64, float64, error)
}

func (impl *meterDecorateMeterCurrentImpl) Currents() (float64, float64, float64, error) {
	return impl.meterCurrent()
}

type meterDecorateMeterEnergyImpl struct {
	meterEnergy func() (float64, error)
}

func (impl *meterDecorateMeterEnergyImpl) TotalEnergy() (float64, error) {
	return impl.meterEnergy()
}
