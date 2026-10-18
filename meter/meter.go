package meter

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/meter/measurement"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

type batteryChargePowerSetter func(float64) error

func (f batteryChargePowerSetter) SetBatteryChargePower(power float64) error {
	return f(power)
}

type batteryChargePowerControl struct {
	batteryChargePowerSetter
	update func(float64) error
}

func (c batteryChargePowerControl) UpdateBatteryChargePower(power float64) error {
	return c.update(power)
}

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates a new meter from config
func NewConfigurableFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	var cc struct {
		measurement.Energy    `mapstructure:",squash"` // energy optional
		measurement.Phases    `mapstructure:",squash"` // optional
		measurement.Dimmer    `mapstructure:",squash"` // optional
		measurement.Curtailer `mapstructure:",squash"` // optional

		// pv
		pvMaxACPowerCtx `mapstructure:",squash"`

		// battery
		batteryCapacityCtx    `mapstructure:",squash"`
		batterySocLimitsCtx   `mapstructure:",squash"`
		batteryPowerLimitsCtx `mapstructure:",squash"`
		Soc                   *plugin.Config // optional
		LimitSoc              *plugin.Config // optional
		BatteryMode           *plugin.Config // optional
		ChargePower           *plugin.Config // optional dynamic charge power setter
		ChargePowerUpdate     *plugin.Config // optional setpoint-only update for owned manual control
		BatteryRestoreMode    *plugin.Config
		BatteryRestoreModes   []string
		BatteryRestoreDefault string
		BatteryControlState   *plugin.Config // optional read-only operating state
		BatteryModes          []string       // optional, modes supported by batteryMode if it cannot report them itself
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.ChargePowerUpdate != nil && cc.ChargePower == nil {
		return nil, errors.New("battery charge power update requires chargepower")
	}

	// default soc limits (nil-preset avoids mapstructure coercing plugin config into the default's type)
	if cc.batterySocLimitsCtx.MinSoc == nil {
		cc.batterySocLimitsCtx.MinSoc = 0
	}
	if cc.batterySocLimitsCtx.MaxSoc == nil {
		cc.batterySocLimitsCtx.MaxSoc = 100
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

		if cc.BatteryControlState != nil {
			get, err := cc.BatteryControlState.StringGetter(ctx)
			if err != nil {
				return nil, fmt.Errorf("battery control state: %w", err)
			}
			implement.Has[api.BatteryControlStateReader](m, &batteryControlStateReader{get: get})
		}

		if cc.BatteryRestoreMode != nil {
			set, err := cc.BatteryRestoreMode.StringSetter(ctx, "nativeMode")
			if err != nil {
				return nil, fmt.Errorf("battery restore mode: %w", err)
			}
			implement.Has[api.BatteryControlRestorer](m, batteryModeRestorer{set: set, modes: cc.BatteryRestoreModes, fallback: cc.BatteryRestoreDefault})
		}
		if cc.ChargePower != nil {
			setPower, err := cc.ChargePower.FloatSetter(ctx, "chargePower")
			if err != nil {
				return nil, fmt.Errorf("battery charge power: %w", err)
			}
			var control api.BatteryChargePowerController = batteryChargePowerSetter(setPower)
			if cc.ChargePowerUpdate != nil {
				update, err := cc.ChargePowerUpdate.FloatSetter(ctx, "chargePower")
				if err != nil {
					return nil, fmt.Errorf("battery charge power update: %w", err)
				}
				control = batteryChargePowerControl{batteryChargePowerSetter(setPower), update}
			}
			implement.Has(m, control)
		}

		// limitSoc expresses normal/hold/charge through the reserve soc (hold uses the live soc),
		// batteryMode switches the device's operating mode. Configured together the limit is written first.
		var limitController func(api.BatteryMode) error
		if cc.LimitSoc != nil {
			limitSocS, err := cc.LimitSoc.FloatSetter(ctx, "limitSoc")
			if err != nil {
				return nil, fmt.Errorf("battery limit soc: %w", err)
			}

			if limitController, err = cc.batterySocLimitsCtx.LimitController(ctx, socG, limitSocS); err != nil {
				return nil, err
			}
		}

		switch {
		case cc.BatteryMode != nil:
			modeS, keys, err := cc.BatteryMode.IntSetterKeys(ctx, "batteryMode")
			if err != nil {
				return nil, fmt.Errorf("battery mode: %w", err)
			}

			// the keys the setter switches on are the modes the battery supports,
			// a setter that cannot report them must declare its modes
			var modes []api.BatteryMode
			if keys != nil {
				modes = batteryModes(keys)
			} else if modes, err = declaredBatteryModes(cc.BatteryModes); err != nil {
				return nil, fmt.Errorf("battery modes: %w", err)
			}

			if len(modes) == 0 {
				return nil, errors.New("battery mode: no supported modes, add batteryModes")
			}

			implement.Has(m, implement.BatteryController(implement.BatteryModes(modes...), func(mode api.BatteryMode) error {
				if limitController != nil && slices.Contains(batteryModesSocLimit(), mode) {
					if err := limitController(mode); err != nil {
						return err
					}
				}
				return modeS(int64(mode))
			}))

		case limitController != nil:
			implement.Has(m, implement.BatteryController(batteryModesSocLimit, limitController))
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

// Only explicitly configured automatic modes may be restored; a GUI decision
// cannot inject arbitrary text into the device's HTTP command.
type batteryModeRestorer struct {
	set      func(string) error
	modes    []string
	fallback string
}

func (r batteryModeRestorer) RestoreBatteryMode(mode string) error {
	if mode == "" {
		mode = r.fallback
	}
	if mode == "" || !slices.Contains(r.modes, mode) {
		return fmt.Errorf("unsupported automatic mode")
	}
	return r.set(mode)
}
