package meter

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add(api.Custom, NewConfigurableFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateMeter -b api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.PhasePowers,Powers,func() (float64, float64, float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryCapacity,Capacity,func() float64" -t "api.BatteryController,SetBatteryMode,func(api.BatteryMode) error"

// NewConfigurableFromConfig creates api.Meter from config
func NewConfigurableFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		Power    provider.Config
		Energy   *provider.Config  // optional
		Currents []provider.Config // optional
		Voltages []provider.Config // optional
		Powers   []provider.Config // optional

		// battery
		capacity    `mapstructure:",squash"`
		battery     `mapstructure:",squash"`
		Soc         *provider.Config // optional
		LimitSoc    *provider.Config // optional
		BatteryMode *provider.Config // optional
	}{
		battery: battery{
			MinSoc: 20,
			MaxSoc: 95,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	powerG, energyG, err := BuildMeasurements(&cc.Power, cc.Energy)
	if err != nil {
		return nil, err
	}

	currentsG, voltagesG, powersG, err := BuildPhaseMeasurements(cc.Currents, cc.Voltages, cc.Powers)
	if err != nil {
		return nil, err
	}

	m, _ := NewConfigurable(powerG)

	// decorate soc
	var socG func() (float64, error)
	if cc.Soc != nil {
		socG, err = provider.NewFloatGetterFromConfig(*cc.Soc)
		if err != nil {
			return nil, fmt.Errorf("battery soc: %w", err)
		}
	}

	var batModeS func(api.BatteryMode) error

	switch {
	case cc.Soc != nil && cc.LimitSoc != nil:
		limitSocS, err := provider.NewFloatSetterFromConfig("limitSoc", *cc.LimitSoc)
		if err != nil {
			return nil, fmt.Errorf("battery limit soc: %w", err)
		}

		batModeS = cc.battery.LimitController(socG, limitSocS)

	case cc.BatteryMode != nil:
		modeS, err := provider.NewIntSetterFromConfig("batteryMode", *cc.BatteryMode)
		if err != nil {
			return nil, fmt.Errorf("battery mode: %w", err)
		}

		batModeS = cc.battery.ModeController(modeS)
	}

	res := m.Decorate(energyG, currentsG, voltagesG, powersG, socG, cc.capacity.Decorator(), batModeS)

	return res, nil
}

// NewConfigurable creates a new meter
func NewConfigurable(currentPowerG func() (float64, error)) (*Meter, error) {
	m := &Meter{
		currentPowerG: currentPowerG,
	}
	return m, nil
}

// Meter is an api.Meter implementation with configurable getters and setters.
type Meter struct {
	currentPowerG func() (float64, error)
}

// Decorate attaches additional capabilities to the base meter
func (m *Meter) Decorate(
	totalEnergy func() (float64, error),
	currents func() (float64, float64, float64, error),
	voltages func() (float64, float64, float64, error),
	powers func() (float64, float64, float64, error),
	batterySoc func() (float64, error),
	capacity func() float64,
	setBatteryMode func(api.BatteryMode) error,
) api.Meter {
	return decorateMeter(m, totalEnergy, currents, voltages, powers, batterySoc, capacity, setBatteryMode)
}

// CurrentPower implements the api.Meter interface
func (m *Meter) CurrentPower() (float64, error) {
	return m.currentPowerG()
}
