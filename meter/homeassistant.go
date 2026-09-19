package meter

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

func init() {
	registry.AddCtx("homeassistant", NewHomeAssistantFromConfig)
}

// NewHomeAssistantFromConfig creates a HomeAssistant meter from generic config
func NewHomeAssistantFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	var cc struct {
		homeassistant.Config `mapstructure:",squash"`
		Power                string
		Energy               string
		ReturnEnergy         string
		Currents             []string
		Voltages             []string
		Powers               []string
		Soc                  string

		// pv
		pvMaxACPower `mapstructure:",squash"`

		// battery
		batteryCapacity     `mapstructure:",squash"`
		batterySocLimitsCtx `mapstructure:",squash"`
		batteryPowerLimits  `mapstructure:",squash"`

		// battery mode control - optional script entities per mode
		ModeNormal     string
		ModeHold       string
		ModeCharge     string
		ModeHoldCharge string
		ModeDischarge  string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// default soc limits (nil-preset avoids mapstructure coercing plugin config into the default's type)
	if cc.batterySocLimitsCtx.MinSoc == nil {
		cc.batterySocLimitsCtx.MinSoc = 0
	}
	if cc.batterySocLimitsCtx.MaxSoc == nil {
		cc.batterySocLimitsCtx.MaxSoc = 100
	}

	if cc.Power == "" {
		return nil, errors.New("missing power sensor entity")
	}

	log := util.NewLogger("ha-meter")

	conn, err := cc.Config.NewConnection(log)
	if err != nil {
		return nil, err
	}

	m, _ := NewConfigurable(func() (float64, error) {
		return conn.GetFloatState(cc.Power)
	})

	// energy
	if cc.Energy != "" {
		implement.Has(m, implement.MeterEnergy(func() (float64, error) {
			return conn.GetFloatState(cc.Energy)
		}))
	}

	if cc.ReturnEnergy != "" {
		implement.Has(m, implement.MeterReturnEnergy(func() (float64, error) {
			return conn.GetFloatState(cc.ReturnEnergy)
		}))
	}

	if cc.Soc != "" {
		socG := func() (float64, error) { return conn.GetFloatState(cc.Soc) }

		socLimiter, err := cc.batterySocLimitsCtx.Decorator(ctx)
		if err != nil {
			return nil, err
		}

		implement.Has(m, implement.Battery(socG))
		implement.May(m, implement.BatteryCapacity(cc.batteryCapacity.Decorator()))
		implement.May(m, implement.BatterySocLimiter(socLimiter))
		implement.May(m, implement.BatteryPowerLimiter(cc.batteryPowerLimits.Decorator()))

		modes := map[api.BatteryMode]string{
			api.BatteryNormal:     cc.ModeNormal,
			api.BatteryHold:       cc.ModeHold,
			api.BatteryCharge:     cc.ModeCharge,
			api.BatteryHoldCharge: cc.ModeHoldCharge,
			api.BatteryDischarge:  cc.ModeDischarge,
		}

		// an unconfigured mode is not supported
		maps.DeleteFunc(modes, func(_ api.BatteryMode, entity string) bool {
			return entity == ""
		})

		if len(modes) > 0 {
			if cc.ModeNormal == "" {
				return nil, errors.New("modeNormal is required when any other battery mode is configured")
			}

			if len(modes) == 1 {
				return nil, errors.New("modeNormal alone has no effect; configure modeHold, modeCharge, modeHoldCharge and/or modeDischarge")
			}

			for _, entity := range modes {
				if !strings.HasPrefix(entity, "script.") {
					return nil, fmt.Errorf("battery mode entity must be a script: %s", entity)
				}
			}

			modeG := implement.BatteryModes(slices.Sorted(maps.Keys(modes))...)
			implement.Has(m, implement.BatteryController(modeG, batteryModeController(conn, modes)))
		}

		return m, nil
	}

	// phase currents (optional)
	if phases, err := homeassistant.ValidatePhaseEntities(cc.Currents); err != nil {
		return nil, fmt.Errorf("currents: %w", err)
	} else if len(phases) > 0 {
		implement.Has(m, implement.PhaseCurrents(func() (float64, float64, float64, error) {
			return conn.GetPhaseFloatStates(phases)
		}))
	}

	// phase voltages (optional)
	if phases, err := homeassistant.ValidatePhaseEntities(cc.Voltages); err != nil {
		return nil, fmt.Errorf("voltages: %w", err)
	} else if len(phases) > 0 {
		implement.Has(m, implement.PhaseVoltages(func() (float64, float64, float64, error) {
			return conn.GetPhaseFloatStates(phases)
		}))
	}

	// phase powers (optional)
	if phases, err := homeassistant.ValidatePhaseEntities(cc.Powers); err != nil {
		return nil, fmt.Errorf("powers: %w", err)
	} else if len(phases) > 0 {
		implement.Has(m, implement.PhasePowers(func() (float64, float64, float64, error) {
			return conn.GetPhaseFloatStates(phases)
		}))
	}

	implement.May(m, implement.MaxACPowerGetter(cc.pvMaxACPower.Decorator()))

	return m, nil
}

// batteryModeController returns a BatteryController function that runs the
// Home Assistant script configured for the requested evcc battery mode. Each
// mode is self-contained: evcc only triggers the matching script and never
// deactivates others - any mutual exclusion is the HA side's responsibility.
// All modes except modeNormal are optional; a mode without a backing script
// is not announced and hence invalid here.
func batteryModeController(conn *homeassistant.Connection, modes map[api.BatteryMode]string) func(api.BatteryMode) error {
	return func(mode api.BatteryMode) error {
		target, ok := modes[mode]
		if !ok {
			return errInvalidBatteryMode(mode)
		}
		return conn.CallSwitchService(target, true)
	}
}
