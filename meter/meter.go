package meter

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/measurement"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

//go:generate go tool decorate -f decorateMeter -b api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.PhasePowers,Powers,func() (float64, float64, float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryCapacity,Capacity,func() float64" -t "api.MaxACPowerGetter,MaxACPower,func() float64" -t "api.BatteryController,SetBatteryMode,func(api.BatteryMode) error"

// NewConfigurableFromConfig creates api.Meter from config
func NewConfigurableFromConfig(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		measurement.Energy `mapstructure:",squash"` // energy optional
		measurement.Phases `mapstructure:",squash"` // optional

		// battery
		capacity    `mapstructure:",squash"`
		maxpower    `mapstructure:",squash"`
		battery     `mapstructure:",squash"`
		Soc         *plugin.Config // optional
		LimitSoc    *plugin.Config // optional
		BatteryMode *plugin.Config // optional
	}{
		battery: battery{
			MinSoc: 20,
			MaxSoc: 95,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	powerG, energyG, err := cc.Energy.Configure(ctx)
	if err != nil {
		return nil, err
	}

	currentsG, voltagesG, powersG, err := cc.Phases.Configure(ctx)
	if err != nil {
		return nil, err
	}

	m, _ := NewConfigurable(powerG)

	// decorate soc
	socG, err := cc.Soc.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("battery soc: %w", err)
	}

	var batModeS func(api.BatteryMode) error

	switch {
	case cc.Soc != nil && cc.LimitSoc != nil:
		limitSocS, err := cc.LimitSoc.FloatSetter(ctx, "limitSoc")
		if err != nil {
			return nil, fmt.Errorf("battery limit soc: %w", err)
		}

		batModeS = cc.battery.LimitController(socG, limitSocS)

	case cc.BatteryMode != nil:
		modeS, err := cc.BatteryMode.IntSetter(ctx, "batteryMode")
		if err != nil {
			return nil, fmt.Errorf("battery mode: %w", err)
		}

		batModeS = func(mode api.BatteryMode) error {
			return modeS(int64(mode))
		}
	}

	res := m.Decorate(energyG, currentsG, voltagesG, powersG, socG, cc.capacity.Decorator(), cc.maxpower.Decorator(), batModeS)

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
	batteryCapacity func() float64,
	maxACPower func() float64,
	setBatteryMode func(api.BatteryMode) error,
) api.Meter {
	return decorateMeter(m, totalEnergy, currents, voltages, powers, batterySoc, batteryCapacity, maxACPower, setBatteryMode)
}

// CurrentPower implements the api.Meter interface
func (m *Meter) CurrentPower() (float64, error) {
	return m.currentPowerG()
}
