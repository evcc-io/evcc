package implement

import (
	"github.com/evcc-io/evcc/api"
)

func Meter(f func() (float64, error)) api.Meter {
	return &iMeter{f}
}

type iMeter struct {
	f func() (float64, error)
}

func (i *iMeter) CurrentPower() (float64, error) {
	return i.f()
}

//

func MeterEnergy(f func() (float64, error)) api.MeterEnergy {
	return &iMeterEnergy{f}
}

type iMeterEnergy struct {
	f func() (float64, error)
}

func (i *iMeterEnergy) TotalEnergy() (float64, error) {
	return i.f()
}

//

func PhaseCurrents(f func() (float64, float64, float64, error)) api.PhaseCurrents {
	return &iPhaseCurrents{f}
}

type iPhaseCurrents struct {
	f func() (float64, float64, float64, error)
}

func (i *iPhaseCurrents) Currents() (float64, float64, float64, error) {
	return i.f()
}

//

func PhaseVoltages(f func() (float64, float64, float64, error)) api.PhaseVoltages {
	return &iPhaseVoltages{f}
}

type iPhaseVoltages struct {
	f func() (float64, float64, float64, error)
}

func (i *iPhaseVoltages) Voltages() (float64, float64, float64, error) {
	return i.f()
}

//

func PhasePowers(f func() (float64, float64, float64, error)) api.PhasePowers {
	return &iPhasePowers{f}
}

type iPhasePowers struct {
	f func() (float64, float64, float64, error)
}

func (i *iPhasePowers) Powers() (float64, float64, float64, error) {
	return i.f()
}

//

func Battery(f func() (float64, error)) api.Battery {
	return &iBattery{f}
}

type iBattery struct {
	f func() (float64, error)
}

func (i *iBattery) Soc() (float64, error) {
	return i.f()
}

//

func BatteryCapacity(f func() float64) api.BatteryCapacity {
	return &iBatteryCapacity{f}
}

type iBatteryCapacity struct {
	f func() float64
}

func (i *iBatteryCapacity) Capacity() float64 {
	return i.f()
}

//

func BatteryPowerLimiter(f func() (float64, float64)) api.BatteryPowerLimiter {
	return &iBatteryPowerLimiter{f}
}

type iBatteryPowerLimiter struct {
	f func() (float64, float64)
}

func (i *iBatteryPowerLimiter) GetPowerLimits() (float64, float64) {
	return i.f()
}

//

func BatterySocLimiter(f func() (float64, float64)) api.BatterySocLimiter {
	return &iBatterySocLimiter{f}
}

type iBatterySocLimiter struct {
	f func() (float64, float64)
}

func (i *iBatterySocLimiter) GetSocLimits() (float64, float64) {
	return i.f()
}

//

func BatteryController(f func(api.BatteryMode) error) api.BatteryController {
	return &iBatteryController{f}
}

type iBatteryController struct {
	f func(api.BatteryMode) error
}

func (i *iBatteryController) SetBatteryMode(m api.BatteryMode) error {
	return i.f(m)
}

//

func MaxACPowerGetter(f func() float64) api.MaxACPowerGetter {
	return &iMaxACPowerGetter{f}
}

type iMaxACPowerGetter struct {
	f func() float64
}

func (i *iMaxACPowerGetter) MaxACPower() float64 {
	return i.f()
}
