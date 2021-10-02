package meter

// Code generated by github.com/evcc-io/evcc/cmd/tools/decorate.go. DO NOT EDIT.

import (
	"github.com/evcc-io/evcc/api"
)

func decorateRCT(base *RCT, meterEnergy func() (float64, error), battery func() (float64, error)) api.Meter {
	switch {
	case battery == nil && meterEnergy == nil:
		return base

	case battery == nil && meterEnergy != nil:
		return &struct {
			*RCT
			api.MeterEnergy
		}{
			RCT: base,
			MeterEnergy: &decorateRCTMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case battery != nil && meterEnergy == nil:
		return &struct {
			*RCT
			api.Battery
		}{
			RCT: base,
			Battery: &decorateRCTBatteryImpl{
				battery: battery,
			},
		}

	case battery != nil && meterEnergy != nil:
		return &struct {
			*RCT
			api.Battery
			api.MeterEnergy
		}{
			RCT: base,
			Battery: &decorateRCTBatteryImpl{
				battery: battery,
			},
			MeterEnergy: &decorateRCTMeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}
	}

	return nil
}

type decorateRCTBatteryImpl struct {
	battery func() (float64, error)
}

func (impl *decorateRCTBatteryImpl) SoC() (float64, error) {
	return impl.battery()
}

type decorateRCTMeterEnergyImpl struct {
	meterEnergy func() (float64, error)
}

func (impl *decorateRCTMeterEnergyImpl) TotalEnergy() (float64, error) {
	return impl.meterEnergy()
}
