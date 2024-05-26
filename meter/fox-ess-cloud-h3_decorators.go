package meter

// Code generated by github.com/evcc-io/evcc/cmd/tools/decorate.go. DO NOT EDIT.

import (
	"github.com/evcc-io/evcc/api"
)

func decorateFoxESSCloudH3(base *FoxESSCloudH3, phasePowers func() (float64, float64, float64, error), meterEnergy func() (float64, error), battery func() (float64, error), batteryController func(api.BatteryMode) error) api.Meter {
	switch {
	case battery == nil && batteryController == nil && meterEnergy == nil && phasePowers == nil:
		return base

	case battery == nil && batteryController == nil && meterEnergy == nil && phasePowers != nil:
		return &struct {
			*FoxESSCloudH3
			api.PhasePowers
		}{
			FoxESSCloudH3: base,
			PhasePowers: &decorateFoxESSCloudH3PhasePowersImpl{
				phasePowers: phasePowers,
			},
		}

	case battery == nil && batteryController == nil && meterEnergy != nil && phasePowers == nil:
		return &struct {
			*FoxESSCloudH3
			api.MeterEnergy
		}{
			FoxESSCloudH3: base,
			MeterEnergy: &decorateFoxESSCloudH3MeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case battery == nil && batteryController == nil && meterEnergy != nil && phasePowers != nil:
		return &struct {
			*FoxESSCloudH3
			api.MeterEnergy
			api.PhasePowers
		}{
			FoxESSCloudH3: base,
			MeterEnergy: &decorateFoxESSCloudH3MeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhasePowers: &decorateFoxESSCloudH3PhasePowersImpl{
				phasePowers: phasePowers,
			},
		}

	case battery != nil && batteryController == nil && meterEnergy == nil && phasePowers == nil:
		return &struct {
			*FoxESSCloudH3
			api.Battery
		}{
			FoxESSCloudH3: base,
			Battery: &decorateFoxESSCloudH3BatteryImpl{
				battery: battery,
			},
		}

	case battery != nil && batteryController == nil && meterEnergy == nil && phasePowers != nil:
		return &struct {
			*FoxESSCloudH3
			api.Battery
			api.PhasePowers
		}{
			FoxESSCloudH3: base,
			Battery: &decorateFoxESSCloudH3BatteryImpl{
				battery: battery,
			},
			PhasePowers: &decorateFoxESSCloudH3PhasePowersImpl{
				phasePowers: phasePowers,
			},
		}

	case battery != nil && batteryController == nil && meterEnergy != nil && phasePowers == nil:
		return &struct {
			*FoxESSCloudH3
			api.Battery
			api.MeterEnergy
		}{
			FoxESSCloudH3: base,
			Battery: &decorateFoxESSCloudH3BatteryImpl{
				battery: battery,
			},
			MeterEnergy: &decorateFoxESSCloudH3MeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case battery != nil && batteryController == nil && meterEnergy != nil && phasePowers != nil:
		return &struct {
			*FoxESSCloudH3
			api.Battery
			api.MeterEnergy
			api.PhasePowers
		}{
			FoxESSCloudH3: base,
			Battery: &decorateFoxESSCloudH3BatteryImpl{
				battery: battery,
			},
			MeterEnergy: &decorateFoxESSCloudH3MeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhasePowers: &decorateFoxESSCloudH3PhasePowersImpl{
				phasePowers: phasePowers,
			},
		}

	case battery == nil && batteryController != nil && meterEnergy == nil && phasePowers == nil:
		return &struct {
			*FoxESSCloudH3
			api.BatteryController
		}{
			FoxESSCloudH3: base,
			BatteryController: &decorateFoxESSCloudH3BatteryControllerImpl{
				batteryController: batteryController,
			},
		}

	case battery == nil && batteryController != nil && meterEnergy == nil && phasePowers != nil:
		return &struct {
			*FoxESSCloudH3
			api.BatteryController
			api.PhasePowers
		}{
			FoxESSCloudH3: base,
			BatteryController: &decorateFoxESSCloudH3BatteryControllerImpl{
				batteryController: batteryController,
			},
			PhasePowers: &decorateFoxESSCloudH3PhasePowersImpl{
				phasePowers: phasePowers,
			},
		}

	case battery == nil && batteryController != nil && meterEnergy != nil && phasePowers == nil:
		return &struct {
			*FoxESSCloudH3
			api.BatteryController
			api.MeterEnergy
		}{
			FoxESSCloudH3: base,
			BatteryController: &decorateFoxESSCloudH3BatteryControllerImpl{
				batteryController: batteryController,
			},
			MeterEnergy: &decorateFoxESSCloudH3MeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case battery == nil && batteryController != nil && meterEnergy != nil && phasePowers != nil:
		return &struct {
			*FoxESSCloudH3
			api.BatteryController
			api.MeterEnergy
			api.PhasePowers
		}{
			FoxESSCloudH3: base,
			BatteryController: &decorateFoxESSCloudH3BatteryControllerImpl{
				batteryController: batteryController,
			},
			MeterEnergy: &decorateFoxESSCloudH3MeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhasePowers: &decorateFoxESSCloudH3PhasePowersImpl{
				phasePowers: phasePowers,
			},
		}

	case battery != nil && batteryController != nil && meterEnergy == nil && phasePowers == nil:
		return &struct {
			*FoxESSCloudH3
			api.Battery
			api.BatteryController
		}{
			FoxESSCloudH3: base,
			Battery: &decorateFoxESSCloudH3BatteryImpl{
				battery: battery,
			},
			BatteryController: &decorateFoxESSCloudH3BatteryControllerImpl{
				batteryController: batteryController,
			},
		}

	case battery != nil && batteryController != nil && meterEnergy == nil && phasePowers != nil:
		return &struct {
			*FoxESSCloudH3
			api.Battery
			api.BatteryController
			api.PhasePowers
		}{
			FoxESSCloudH3: base,
			Battery: &decorateFoxESSCloudH3BatteryImpl{
				battery: battery,
			},
			BatteryController: &decorateFoxESSCloudH3BatteryControllerImpl{
				batteryController: batteryController,
			},
			PhasePowers: &decorateFoxESSCloudH3PhasePowersImpl{
				phasePowers: phasePowers,
			},
		}

	case battery != nil && batteryController != nil && meterEnergy != nil && phasePowers == nil:
		return &struct {
			*FoxESSCloudH3
			api.Battery
			api.BatteryController
			api.MeterEnergy
		}{
			FoxESSCloudH3: base,
			Battery: &decorateFoxESSCloudH3BatteryImpl{
				battery: battery,
			},
			BatteryController: &decorateFoxESSCloudH3BatteryControllerImpl{
				batteryController: batteryController,
			},
			MeterEnergy: &decorateFoxESSCloudH3MeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
		}

	case battery != nil && batteryController != nil && meterEnergy != nil && phasePowers != nil:
		return &struct {
			*FoxESSCloudH3
			api.Battery
			api.BatteryController
			api.MeterEnergy
			api.PhasePowers
		}{
			FoxESSCloudH3: base,
			Battery: &decorateFoxESSCloudH3BatteryImpl{
				battery: battery,
			},
			BatteryController: &decorateFoxESSCloudH3BatteryControllerImpl{
				batteryController: batteryController,
			},
			MeterEnergy: &decorateFoxESSCloudH3MeterEnergyImpl{
				meterEnergy: meterEnergy,
			},
			PhasePowers: &decorateFoxESSCloudH3PhasePowersImpl{
				phasePowers: phasePowers,
			},
		}
	}

	return nil
}

type decorateFoxESSCloudH3BatteryImpl struct {
	battery func() (float64, error)
}

func (impl *decorateFoxESSCloudH3BatteryImpl) Soc() (float64, error) {
	return impl.battery()
}

type decorateFoxESSCloudH3BatteryControllerImpl struct {
	batteryController func(api.BatteryMode) error
}

func (impl *decorateFoxESSCloudH3BatteryControllerImpl) SetBatteryMode(p0 api.BatteryMode) error {
	return impl.batteryController(p0)
}

type decorateFoxESSCloudH3MeterEnergyImpl struct {
	meterEnergy func() (float64, error)
}

func (impl *decorateFoxESSCloudH3MeterEnergyImpl) TotalEnergy() (float64, error) {
	return impl.meterEnergy()
}

type decorateFoxESSCloudH3PhasePowersImpl struct {
	phasePowers func() (float64, float64, float64, error)
}

func (impl *decorateFoxESSCloudH3PhasePowersImpl) Powers() (float64, float64, float64, error) {
	return impl.phasePowers()
}
