package meter

// This file has been generated - do not modify

import (
	"github.com/andig/evcc/api"
)

func decorate(base api.Meter, meterEnergy func() (float64, error), meterCurrent func() (float64, float64, float64, error)) api.Meter {
	switch {
	case meterCurrent == nil && meterEnergy == nil:
		return base

	case meterCurrent == nil && meterEnergy != nil:
		return &struct {
			api.Meter
			api.MeterEnergy
		}{
			Meter: base,
			MeterEnergy: &meterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meterCurrent != nil && meterEnergy == nil:
		return &struct {
			api.Meter
			api.MeterCurrent
		}{
			Meter: base,
			MeterCurrent: &meterCurrentImpl{
				meterCurrent: meterCurrent,
			},
		}

	case meterCurrent != nil && meterEnergy != nil:
		return &struct {
			api.Meter
			api.MeterCurrent
			api.MeterEnergy
		}{
			Meter: base,
			MeterCurrent: &meterCurrentImpl{
				meterCurrent: meterCurrent,
			},
			MeterEnergy: &meterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}
	}

	return nil
}

type meterCurrentImpl struct {
	meterCurrent func() (float64, float64, float64, error)
}

func (impl *meterCurrentImpl) Currents() (float64, float64, float64, error) {
	return impl.meterCurrent()
}

type meterEnergyImpl struct {
	meterEnergy func() (float64, error)
}

func (impl *meterEnergyImpl) TotalEnergy() (float64, error) {
	return impl.meterEnergy()
}
