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
			MeterEnergy: &decorateMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meterCurrent != nil && meterEnergy == nil:
		return &struct{
			api.Charger
			api.MeterCurrent
		}{
			Charger: base,
			MeterCurrent: &decorateMeterCurrentImpl{
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
