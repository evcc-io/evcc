package charger

// Code generated by github.com/evcc-io/evcc/cmd/tools/decorate.go. DO NOT EDIT.

import (
	"github.com/evcc-io/evcc/api"
)

func decorateWarp(base *Warp, meter func() (float64, error), meterEnergy func() (float64, error), meterCurrent func() (float64, float64, float64, error)) api.Charger {
	switch {
	case meter == nil && meterCurrent == nil && meterEnergy == nil:
		return base

	case meter != nil && meterCurrent == nil && meterEnergy == nil:
		return &struct {
			*Warp
			api.Meter
		}{
			Warp: base,
			Meter: &decorateWarpMeterImpl{
				meter: meter,
			},
		}

	case meter == nil && meterCurrent == nil && meterEnergy != nil:
		return &struct {
			*Warp
			api.MeterEnergy
		}{
			Warp: base,
			MeterEnergy: &decorateWarpMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meter != nil && meterCurrent == nil && meterEnergy != nil:
		return &struct {
			*Warp
			api.Meter
			api.MeterEnergy
		}{
			Warp: base,
			Meter: &decorateWarpMeterImpl{
				meter: meter,
			},
			MeterEnergy: &decorateWarpMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meter == nil && meterCurrent != nil && meterEnergy == nil:
		return &struct {
			*Warp
			api.MeterCurrent
		}{
			Warp: base,
			MeterCurrent: &decorateWarpMeterCurrentImpl{
				meterCurrent: meterCurrent,
			},
		}

	case meter != nil && meterCurrent != nil && meterEnergy == nil:
		return &struct {
			*Warp
			api.Meter
			api.MeterCurrent
		}{
			Warp: base,
			Meter: &decorateWarpMeterImpl{
				meter: meter,
			},
			MeterCurrent: &decorateWarpMeterCurrentImpl{
				meterCurrent: meterCurrent,
			},
		}

	case meter == nil && meterCurrent != nil && meterEnergy != nil:
		return &struct {
			*Warp
			api.MeterCurrent
			api.MeterEnergy
		}{
			Warp: base,
			MeterCurrent: &decorateWarpMeterCurrentImpl{
				meterCurrent: meterCurrent,
			},
			MeterEnergy: &decorateWarpMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case meter != nil && meterCurrent != nil && meterEnergy != nil:
		return &struct {
			*Warp
			api.Meter
			api.MeterCurrent
			api.MeterEnergy
		}{
			Warp: base,
			Meter: &decorateWarpMeterImpl{
				meter: meter,
			},
			MeterCurrent: &decorateWarpMeterCurrentImpl{
				meterCurrent: meterCurrent,
			},
			MeterEnergy: &decorateWarpMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}
	}

	return nil
}

type decorateWarpMeterImpl struct {
	meter func() (float64, error)
}

func (impl *decorateWarpMeterImpl) CurrentPower() (float64, error) {
	return impl.meter()
}

type decorateWarpMeterCurrentImpl struct {
	meterCurrent func() (float64, float64, float64, error)
}

func (impl *decorateWarpMeterCurrentImpl) Currents() (float64, float64, float64, error) {
	return impl.meterCurrent()
}

type decorateWarpMeterEnergyImpl struct {
	meterEnergy func() (float64, error)
}

func (impl *decorateWarpMeterEnergyImpl) TotalEnergy() (float64, error) {
	return impl.meterEnergy()
}
