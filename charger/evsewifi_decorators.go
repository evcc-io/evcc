package charger

// This file has been generated - do not modify

import (
	"github.com/andig/evcc/api"
)

func decorateEVSE(base api.Charger, meterEnergy func() (float64, error), meterCurrent func() (float64, float64, float64, error)) api.Charger {
	switch {
	case meterCurrent == nil && meterEnergy == nil:
		return base

	case meterCurrent == nil && meterEnergy != nil:
		return &struct{
			api.Charger
			api.MeterEnergy
		}{
			Charger: base,
			MeterEnergy: &decorateEVSEMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meterCurrent != nil && meterEnergy == nil:
		return &struct{
			api.Charger
			api.MeterCurrent
		}{
			Charger: base,
			MeterCurrent: &decorateEVSEMeterCurrentImpl{
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
			MeterCurrent: &decorateEVSEMeterCurrentImpl{
				meterCurrent: meterCurrent,
			},
			MeterEnergy: &decorateEVSEMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}
	}

	return nil
}

type decorateEVSEMeterCurrentImpl struct {
	meterCurrent func() (float64, float64, float64, error)
}

func (impl *decorateEVSEMeterCurrentImpl) Currents() (float64, float64, float64, error) {
	return impl.meterCurrent()
}

type decorateEVSEMeterEnergyImpl struct {
	meterEnergy func() (float64, error)
}

func (impl *decorateEVSEMeterEnergyImpl) TotalEnergy() (float64, error) {
	return impl.meterEnergy()
}
