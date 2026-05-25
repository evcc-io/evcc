package meter

import (
	"errors"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

func init() {
	registry.Add("homeassistant", NewHomeAssistantFromConfig)
}

// NewHomeAssistantFromConfig creates a HomeAssistant meter from generic config
func NewHomeAssistantFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		URI      string
		Token_   string `mapstructure:"token"` // TODO deprecated
		Home_    string `mapstructure:"home"`  // TODO deprecated
		Power    string
		Energy   string
		Currents []string
		Voltages []string
		Powers   []string
		Soc      string

		// pv
		pvMaxACPower `mapstructure:",squash"`

		// battery
		batteryCapacity    `mapstructure:",squash"`
		batterySocLimits   `mapstructure:",squash"`
		batteryPowerLimits `mapstructure:",squash"`

		// battery mode control - optional switch-like entities per mode
		ModeNormal string
		ModeHold   string
		ModeCharge string
	}{
		batterySocLimits: batterySocLimits{
			MinSoc: 20,
			MaxSoc: 95,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Power == "" {
		return nil, errors.New("missing power sensor entity")
	}

	log := util.NewLogger("ha-meter")

	conn, err := homeassistant.NewConnection(log, cc.URI, cc.Home_)
	if err != nil {
		return nil, err
	}

	m, _ := NewConfigurable(func() (float64, error) {
		return conn.GetFloatState(cc.Power)
	})

	// decorators for optional interfaces
	var energyG func() (float64, error)
	var currentsG, voltagesG, powersG func() (float64, float64, float64, error)

	if cc.Energy != "" {
		energyG = func() (float64, error) { return conn.GetFloatState(cc.Energy) }
	}

	// phase currents (optional)
	if phases, err := homeassistant.ValidatePhaseEntities(cc.Currents); err != nil {
		return nil, fmt.Errorf("currents: %w", err)
	} else if len(phases) > 0 {
		currentsG = func() (float64, float64, float64, error) { return conn.GetPhaseFloatStates(phases) }
	}

	// phase voltages (optional)
	if phases, err := homeassistant.ValidatePhaseEntities(cc.Voltages); err != nil {
		return nil, fmt.Errorf("voltages: %w", err)
	} else if len(phases) > 0 {
		voltagesG = func() (float64, float64, float64, error) { return conn.GetPhaseFloatStates(phases) }
	}

	// phase powers (optional)
	if phases, err := homeassistant.ValidatePhaseEntities(cc.Powers); err != nil {
		return nil, fmt.Errorf("powers: %w", err)
	} else if len(phases) > 0 {
		powersG = func() (float64, float64, float64, error) { return conn.GetPhaseFloatStates(phases) }
	}

	implement.May(m, implement.MeterEnergy(energyG))

	if cc.Soc != "" {
		socG := func() (float64, error) { return conn.GetFloatState(cc.Soc) }

		implement.Has(m, implement.Battery(socG))
		implement.May(m, implement.BatteryCapacity(cc.batteryCapacity.Decorator()))
		implement.May(m, implement.BatterySocLimiter(cc.batterySocLimits.Decorator()))
		implement.May(m, implement.BatteryPowerLimiter(cc.batteryPowerLimits.Decorator()))

		if cc.ModeHold != "" || cc.ModeCharge != "" {
			if cc.ModeNormal == "" {
				return nil, errors.New("modeNormal is required when modeHold or modeCharge is configured")
			}
			modes := map[api.BatteryMode]string{
				api.BatteryNormal: cc.ModeNormal,
				api.BatteryHold:   cc.ModeHold,
				api.BatteryCharge: cc.ModeCharge,
			}
			for _, entity := range modes {
				if entity != "" && !strings.HasPrefix(entity, "script.") {
					return nil, fmt.Errorf("battery mode entity must be a script: %s", entity)
				}
			}
			implement.Has(m, implement.BatteryController(batteryModeController(conn, modes)))
		} else if cc.ModeNormal != "" {
			return nil, errors.New("modeNormal alone has no effect; configure modeHold and/or modeCharge")
		}

		return m, nil
	}

	implement.May(m, implement.PhaseCurrents(currentsG))
	implement.May(m, implement.PhaseVoltages(voltagesG))
	implement.May(m, implement.PhasePowers(powersG))
	implement.May(m, implement.MaxACPowerGetter(cc.pvMaxACPower.Decorator()))

	return m, nil
}

// batteryModeController returns a BatteryController function that activates
// the switch-like Home Assistant entity configured for the requested evcc
// battery mode. Each mode is self-contained: evcc only triggers the matching
// entity and never deactivates others - any mutual exclusion is the HA side's
// responsibility. modeHold and modeCharge are optional and return
// api.ErrNotAvailable when requested without a backing entity.
func batteryModeController(conn *homeassistant.Connection, modes map[api.BatteryMode]string) func(api.BatteryMode) error {
	return func(mode api.BatteryMode) error {
		target, ok := modes[mode]
		if !ok || target == "" {
			return api.ErrNotAvailable
		}
		return conn.CallSwitchService(target, true)
	}
}
