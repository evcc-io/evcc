package meter

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
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
		Home     string // TODO deprecated
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

	conn, err := homeassistant.NewConnection(log, cc.URI, cc.Home)
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
	if phases, err := homeassistant.ValidatePhaseEntities(cc.Currents); len(phases) > 0 {
		currentsG = func() (float64, float64, float64, error) { return conn.GetPhaseFloatStates(phases) }
	} else if err != nil {
		return nil, fmt.Errorf("currents: %w", err)
	}

	// phase voltages (optional)
	if phases, err := homeassistant.ValidatePhaseEntities(cc.Voltages); len(phases) > 0 {
		voltagesG = func() (float64, float64, float64, error) { return conn.GetPhaseFloatStates(phases) }
	} else if err != nil {
		return nil, fmt.Errorf("voltages: %w", err)
	}

	// phase powers (optional)
	if phases, err := homeassistant.ValidatePhaseEntities(cc.Powers); len(phases) > 0 {
		powersG = func() (float64, float64, float64, error) { return conn.GetPhaseFloatStates(phases) }
	} else if err != nil {
		return nil, fmt.Errorf("powers: %w", err)
	}

	if cc.Soc != "" {
		socG := func() (float64, error) { return conn.GetFloatState(cc.Soc) }

		return m.DecorateBattery(
			energyG,
			socG, cc.batteryCapacity.Decorator(),
			cc.batterySocLimits.Decorator(), cc.batteryPowerLimits.Decorator(),
			nil,
		), nil
	}

	return m.Decorate(
		energyG, currentsG, voltagesG, powersG, cc.pvMaxACPower.Decorator(),
	), nil
}
