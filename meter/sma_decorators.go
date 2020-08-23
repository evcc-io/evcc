package meter

// This file has been generated - do not modify

import (
	"github.com/andig/evcc/api"
)

func smaDecorate(base api.Meter, meterEnergy func() (float64, error)) api.Meter {
	switch {
	case meterEnergy == nil:
		return base

	case meterEnergy != nil:
		return &struct{
			api.Meter
			api.MeterEnergy
		}{
			Meter: base,
			MeterEnergy: &smaDecorateMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}
	}

	return nil
}

type smaDecorateMeterEnergyImpl struct {
	meterEnergy func() (float64, error)
}

func (impl *smaDecorateMeterEnergyImpl) TotalEnergy() (float64, error) {
	return impl.meterEnergy()
}
