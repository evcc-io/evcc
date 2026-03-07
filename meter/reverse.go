package meter

import "github.com/evcc-io/evcc/api"

func reverseFloatGetter(g func() (float64, error)) func() (float64, error) {
	if g == nil {
		return nil
	}

	return func() (float64, error) {
		f, err := g()
		return -f, err
	}
}

func reversePhaseGetter(g func() (float64, float64, float64, error)) func() (float64, float64, float64, error) {
	if g == nil {
		return nil
	}

	return func() (float64, float64, float64, error) {
		l1, l2, l3, err := g()
		return -l1, -l2, -l3, err
	}
}

func Reverse(m api.Meter) api.Meter {
	meter, _ := NewConfigurable(reverseFloatGetter(m.CurrentPower))

	var totalEnergy func() (float64, error)
	if energyMeter, ok := m.(api.MeterEnergy); ok {
		totalEnergy = energyMeter.TotalEnergy
	}

	var currents func() (float64, float64, float64, error)
	if phaseMeter, ok := m.(api.PhaseCurrents); ok {
		currents = reversePhaseGetter(phaseMeter.Currents)
	}

	var voltages func() (float64, float64, float64, error)
	if phaseMeter, ok := m.(api.PhaseVoltages); ok {
		voltages = phaseMeter.Voltages
	}

	var powers func() (float64, float64, float64, error)
	if phaseMeter, ok := m.(api.PhasePowers); ok {
		powers = reversePhaseGetter(phaseMeter.Powers)
	}

	var maxACPower func() float64
	if pvMeter, ok := m.(api.MaxACPowerGetter); ok {
		maxACPower = pvMeter.MaxACPower
	}

	var soc func() (float64, error)
	if battery, ok := m.(api.Battery); ok {
		soc = battery.Soc
	}

	var capacity func() float64
	if battery, ok := m.(api.BatteryCapacity); ok {
		capacity = battery.Capacity
	}

	var socLimits func() (float64, float64)
	if battery, ok := m.(api.BatterySocLimiter); ok {
		socLimits = battery.GetSocLimits
	}

	var powerLimits func() (float64, float64)
	if battery, ok := m.(api.BatteryPowerLimiter); ok {
		powerLimits = battery.GetPowerLimits
	}

	var setMode func(api.BatteryMode) error
	if battery, ok := m.(api.BatteryController); ok {
		setMode = battery.SetBatteryMode
	}

	if soc != nil {
		return meter.DecorateBattery(totalEnergy, soc, capacity, socLimits, powerLimits, setMode)
	}

	return meter.Decorate(totalEnergy, currents, voltages, powers, maxACPower)
}
