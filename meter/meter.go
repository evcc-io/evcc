package meter

import (
	"context"
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/meter/measurement"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates a new meter from config
func NewConfigurableFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		measurement.Energy    `mapstructure:",squash"` // energy optional
		measurement.Phases    `mapstructure:",squash"` // optional
		measurement.Dimmer    `mapstructure:",squash"` // optional
		measurement.Curtailer `mapstructure:",squash"` // optional

		// pv
		pvMaxACPowerCtx `mapstructure:",squash"`

		// battery
		batteryCapacityCtx     `mapstructure:",squash"`
		batterySocLimitsCtx    `mapstructure:",squash"`
		batteryPowerLimitsCtx  `mapstructure:",squash"`
		Soc                    *plugin.Config // optional
		LimitSoc               *plugin.Config // optional
		BatteryMode            *plugin.Config // optional
		LimitChargePower       *plugin.Config // optional
		LimitChargePowerEnable *plugin.Config // optional, see wiring below
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// default soc limits (nil-preset avoids mapstructure coercing plugin config into the default's type)
	if cc.batterySocLimitsCtx.MinSoc == nil {
		cc.batterySocLimitsCtx.MinSoc = 20
	}
	if cc.batterySocLimitsCtx.MaxSoc == nil {
		cc.batterySocLimitsCtx.MaxSoc = 95
	}

	powerG, energyG, returnG, err := cc.Energy.Configure(ctx)
	if err != nil {
		return nil, err
	}

	m, _ := NewConfigurable(powerG)
	implement.May(m, implement.MeterEnergy(energyG))
	implement.May(m, implement.MeterReturnEnergy(returnG))

	// dim/curtail
	if err := cc.Dimmer.Implement(ctx, m); err != nil {
		return nil, err
	}
	if err := cc.Curtailer.Implement(ctx, m); err != nil {
		return nil, err
	}

	// decorate soc
	socG, err := cc.Soc.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("battery soc: %w", err)
	}

	if socG != nil {
		capacity, err := cc.batteryCapacityCtx.Decorator(ctx)
		if err != nil {
			return nil, err
		}

		socLimiter, err := cc.batterySocLimitsCtx.Decorator(ctx)
		if err != nil {
			return nil, err
		}

		powerLimiter, err := cc.batteryPowerLimitsCtx.Decorator(ctx)
		if err != nil {
			return nil, err
		}

		implement.Has(m, implement.Battery(socG))
		implement.May(m, implement.BatteryCapacity(capacity))
		implement.May(m, implement.BatterySocLimiter(socLimiter))
		implement.May(m, implement.BatteryPowerLimiter(powerLimiter))

		switch {
		case cc.Soc != nil && cc.LimitSoc != nil:
			limitSocS, err := cc.LimitSoc.FloatSetter(ctx, "limitSoc")
			if err != nil {
				return nil, fmt.Errorf("battery limit soc: %w", err)
			}

			limitController, err := cc.batterySocLimitsCtx.LimitController(ctx, socG, limitSocS)
			if err != nil {
				return nil, err
			}

			implement.Has(m, implement.BatteryController(limitController))

		case cc.BatteryMode != nil:
			modeS, err := cc.BatteryMode.IntSetter(ctx, "batteryMode")
			if err != nil {
				return nil, fmt.Errorf("battery mode: %w", err)
			}

			implement.Has(m, implement.BatteryController(func(mode api.BatteryMode) error {
				return modeS(int64(mode))
			}))
		}

		// charge power limit is orthogonal to mode/limit-soc control, hence not part of the switch above
		if cc.LimitChargePower != nil {
			maxChargePowerG, err := resolveFloat(ctx, cc.batteryPowerLimitsCtx.MaxChargePower)
			if err != nil {
				return nil, err
			}
			if maxChargePowerG == nil {
				return nil, errors.New("battery charge power limit requires maxchargepower")
			}

			limitChargePowerS, err := cc.LimitChargePower.FloatSetter(ctx, "limitChargePower")
			if err != nil {
				return nil, fmt.Errorf("battery charge power limit: %w", err)
			}

			// LimitChargePowerEnable is optional: some devices (e.g. SunSpec model 124, where the
			// rate register also governs discharge) need an explicit enable/disable gate so releasing
			// the limit can fully disengage rate control rather than merely writing the max value.
			// Devices with a genuinely standalone charge-power register can omit it.
			var enableS func(int64) error
			if cc.LimitChargePowerEnable != nil {
				enableS, err = cc.LimitChargePowerEnable.IntSetter(ctx, "limitChargePowerEnable")
				if err != nil {
					return nil, fmt.Errorf("battery charge power limit enable: %w", err)
				}
			}

			implement.Has(m, implement.BatteryChargePowerLimiter(func(power *float64) error {
				if power == nil {
					if enableS != nil {
						return enableS(0)
					}
					return limitChargePowerS(maxChargePowerG())
				}

				if enableS != nil {
					if err := enableS(1); err != nil {
						return err
					}
				}

				return limitChargePowerS(min(max(*power, 0), maxChargePowerG()))
			}))
		}

		return m, nil
	}

	currentsG, voltagesG, powersG, err := cc.Phases.Configure(ctx)
	if err != nil {
		return nil, err
	}

	implement.May(m, implement.PhaseCurrents(currentsG))
	implement.May(m, implement.PhaseVoltages(voltagesG))
	implement.May(m, implement.PhasePowers(powersG))
	implement.May(m, implement.MaxACPowerGetter(cc.pvMaxACPowerCtx.Decorator(ctx)))

	return m, nil
}

// NewConfigurable creates a new meter
func NewConfigurable(currentPowerG func() (float64, error)) (*Meter, error) {
	m := &Meter{
		Caps:          implement.New(),
		currentPowerG: currentPowerG,
	}
	return m, nil
}

// Meter is an api.Meter implementation with configurable getters and setters.
type Meter struct {
	implement.Caps
	currentPowerG func() (float64, error)
}

// CurrentPower implements the api.Meter interface
func (m *Meter) CurrentPower() (float64, error) {
	return m.currentPowerG()
}
