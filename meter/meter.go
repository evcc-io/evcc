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

//go:generate go tool decorate

//evcc:function decorateMeter
//evcc:basetype api.Meter
//evcc:types api.MeterEnergy,api.PhaseCurrents,api.PhaseVoltages,api.PhasePowers,api.MaxACPowerGetter

//evcc:function decorateMeterBattery
//evcc:basetype api.Meter
//evcc:types api.MeterEnergy,api.Battery,api.BatteryCapacity,api.BatterySocLimiter,api.BatteryPowerLimiter,api.BatteryController

// NewConfigurableFromConfig creates api.Meter from config
func NewConfigurableFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		measurement.Energy `mapstructure:",squash"` // energy optional
		measurement.Phases `mapstructure:",squash"` // optional

		// pv
		pvMaxACPower `mapstructure:",squash"`

		// battery
		batteryCapacity    `mapstructure:",squash"`
		batterySocLimits   `mapstructure:",squash"`
		batteryPowerLimits `mapstructure:",squash"`
		Soc                *plugin.Config // optional
		LimitSoc           *plugin.Config // optional
		BatteryMode        *plugin.Config // optional
	}{
		batterySocLimits: batterySocLimits{
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

		batModeS = cc.batterySocLimits.LimitController(socG, limitSocS)

	case cc.BatteryMode != nil:
		modeS, err := cc.BatteryMode.IntSetter(ctx, "batteryMode")
		if err != nil {
			return nil, fmt.Errorf("battery mode: %w", err)
		}

		batModeS = func(mode api.BatteryMode) error {
			return modeS(int64(mode))
		}
	}

	if socG != nil {
		return m.DecorateBattery(
			energyG,
			socG, cc.batteryCapacity.Decorator(),
			cc.batterySocLimits.Decorator(), cc.batteryPowerLimits.Decorator(),
			batModeS,
		), nil
	}

	return m.Decorate(
		energyG, currentsG, voltagesG, powersG, cc.pvMaxACPower.Decorator(),
	), nil
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
	currents, voltages, powers func() (float64, float64, float64, error),
	maxACPower func() float64,
) api.Meter {
	return decorateMeter(m,
		totalEnergy, currents, voltages, powers,
		maxACPower,
	)
}

func (m *Meter) DecorateBattery(
	totalEnergy func() (float64, error),
	soc func() (float64, error), capacity func() float64,
	socLimits, powerLimits func() (float64, float64),
	setMode func(api.BatteryMode) error,
) api.Meter {
	return decorateMeterBattery(m,
		totalEnergy,
		soc, capacity,
		socLimits, powerLimits,
		setMode,
	)
}

// CurrentPower implements the api.Meter interface
func (m *Meter) CurrentPower() (float64, error) {
	return m.currentPowerG()
}
