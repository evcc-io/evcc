package charger

//go:generate go tool decorate -f decorateHomeAssistant -b *HomeAssistant -r api.Charger -t api.Meter,api.MeterEnergy,api.PhaseCurrents,api.PhaseVoltages,api.PhaseSwitcher,api.PhaseGetter
//  -t api.CurrentGetter

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

// HomeAssistant charger implementation
type HomeAssistant struct {
	conn       *homeassistant.Connection
	status     string
	enabled    string
	enable     string
	maxcurrent string
}

func init() {
	registry.Add("homeassistant", NewHomeAssistantFromConfig)
}

// NewHomeAssistantFromConfig creates a HomeAssistant charger from generic config
func NewHomeAssistantFromConfig(other map[string]any) (api.Charger, error) {
	var cc struct {
		URI        string
		Token_     string   `mapstructure:"token"` // TODO deprecated
		Home_      string   `mapstructure:"home"`  // TODO deprecated
		Status     string   // required - sensor for charge status
		Enabled    string   // required - sensor for enabled state
		Enable     string   // required - switch/input_boolean for enable/disable
		MaxCurrent string   // required - number entity for setting max current
		Power      string   // optional - power sensor
		Energy     string   // optional - energy sensor
		Currents   []string // optional - current sensors for L1, L2, L3
		Voltages   []string // optional - voltage sensors for L1, L2, L3
		Phases     string   // optional - select entity for 1p/3p phase switching
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Status == "" {
		return nil, errors.New("missing status sensor entity")
	}
	if cc.Enabled == "" {
		return nil, errors.New("missing enabled sensor entity")
	}
	if cc.Enable == "" {
		return nil, errors.New("missing enable switch entity")
	}
	if cc.MaxCurrent == "" {
		return nil, errors.New("missing maxcurrent number entity")
	}

	log := util.NewLogger("ha-charger")

	conn, err := homeassistant.NewConnection(log, cc.URI, cc.Home_)
	if err != nil {
		return nil, err
	}

	c := &HomeAssistant{
		conn:       conn,
		status:     cc.Status,
		enabled:    cc.Enabled,
		enable:     cc.Enable,
		maxcurrent: cc.MaxCurrent,
	}

	// decorators for optional interfaces
	var power, energy func() (float64, error)
	var currents, voltages func() (float64, float64, float64, error)
	var phases1p3p func(int) error
	var phasesG func() (int, error)

	if cc.Power != "" {
		power = func() (float64, error) { return conn.GetFloatState(cc.Power) }
	}
	if cc.Energy != "" {
		energy = func() (float64, error) { return conn.GetFloatState(cc.Energy) }
	}

	// phase currents (optional)
	if phases, err := homeassistant.ValidatePhaseEntities(cc.Currents); len(phases) > 0 {
		currents = func() (float64, float64, float64, error) { return conn.GetPhaseFloatStates(phases) }
	} else if err != nil {
		return nil, fmt.Errorf("currents: %w", err)
	}

	// phase voltages (optional)
	if phases, err := homeassistant.ValidatePhaseEntities(cc.Voltages); len(phases) > 0 {
		voltages = func() (float64, float64, float64, error) { return conn.GetPhaseFloatStates(phases) }
	} else if err != nil {
		return nil, fmt.Errorf("voltages: %w", err)
	}

	// phase switching (optional)
	if cc.Phases != "" {
		phases1p3p = func(phases int) error {
			return conn.CallSelectService(cc.Phases, strconv.Itoa(phases))
		}

		phasesG = func() (int, error) {
			val, err := conn.GetIntState(cc.Phases)
			if err != nil {
				return 0, err
			}
			return int(val), nil
		}
	}

	return decorateHomeAssistant(c, power, energy, currents, voltages, phases1p3p, phasesG), nil
}

var _ api.Charger = (*HomeAssistant)(nil)

// Status implements the api.ChargeState interface
func (c *HomeAssistant) Status() (api.ChargeStatus, error) {
	return c.conn.GetChargeStatus(c.status)
}

// Enabled implements the api.Charger interface
func (c *HomeAssistant) Enabled() (bool, error) {
	return c.conn.GetBoolState(c.enabled)
}

// Enable implements the api.Charger interface
func (c *HomeAssistant) Enable(enable bool) error {
	return c.conn.CallSwitchService(c.enable, enable)
}

// MaxCurrent implements the api.Charger interface
func (c *HomeAssistant) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*HomeAssistant)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *HomeAssistant) MaxCurrentMillis(current float64) error {
	return c.conn.CallNumberService(c.maxcurrent, current)
}
