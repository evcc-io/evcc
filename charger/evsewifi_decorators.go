package charger

// This file has been generated - do not modify

import (
	"github.com/andig/evcc/api"
)

func decorate(base api.Charger, meterEnergy func() (float64, error), meterCurrent func() (float64, float64, float64, error)) api.Charger {
	switch {
	case meterCurrent == nil && meterEnergy == nil:
		return base

	case meterCurrent == nil && meterEnergy != nil:
		return &struct{
			api.Charger
			api.MeterEnergy
		}{
			Charger: base,
			MeterEnergy: &meterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meterCurrent != nil && meterEnergy == nil:
		return &struct{
			api.Charger
			api.MeterCurrent
		}{
			Charger: base,
			MeterCurrent: &meterCurrentImpl{
				meterCurrent: meterCurrent,
			},
		}

	case meterCurrent != nil && meterEnergy != nil:
		return &struct{
			api.Charger
			api.MeterCurrent
			api.MeterEnergy
		}{
			Charger: base,
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
