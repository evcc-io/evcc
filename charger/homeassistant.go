package charger

//go:generate go tool decorate -f decorateHomeAssistant -b *HomeAssistant -r api.Charger -t api.Meter,api.MeterEnergy,api.PhaseCurrents,api.PhaseVoltages, api.PhaseSwitcher,api.PhaseGetter
//  -t api.CurrentGetter

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

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
	phases	   string // Option- select entity for 1p/3p phase switching 
}

func init() {
	registry.Add("homeassistant", NewHomeAssistantFromConfig)
}

// NewHomeAssistantFromConfig creates a HomeAssistant charger from generic config
func NewHomeAssistantFromConfig(other map[string]any) (api.Charger, error) {
	var cc struct {
		URI        string
		Token_     string   `mapstructure:"token"` // TODO deprecated
		Home       string   // TODO deprecated
		Status     string   // required - sensor for charge status
		Enabled    string   // required - sensor for enabled state
		Enable     string   // required - switch/input_boolean for enable/disable
		MaxCurrent string   // required - number entity for setting max current
		Power      string   // optional - power sensor
		Energy     string   // optional - energy sensor
		Currents   []string // optional - current sensors for L1, L2, L3
		Voltages   []string // optional - voltage sensors for L1, L2, L3
		Phases     string   // NEW optional - select entity for 1p/3p phase switching 
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

	conn, err := homeassistant.NewConnection(log, cc.URI, cc.Home)
	if err != nil {
		return nil, err
	}

	c := &HomeAssistant{
		conn:       conn,
		status:     cc.Status,
		enabled:    cc.Enabled,
		enable:     cc.Enable,
		maxcurrent: cc.MaxCurrent,
		phases: 	cc.Phases, 
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

	if cc.Phases != "" {
		phases1p3p = c.phases1p3p
		phasesG = c.getPhases
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

// Phase Switching implements the api.PhaseSwitcher interface
func (c *HomeAssistant) phases1p3p(phases int) error {
	if c.phases == "" {
		return errors.New("phase switching not configured")
	}

	// set phase select entity (e.g. select.wallbox_phases -> "1" or "3")
	option := strconv.Itoa(phases)
	
	// 1) Set phase select entity (e.g. select.wallbox_phases -> "1" or "3")
	if err := c.conn.CallSelectService(c.phases, option); err != nil {
		return fmt.Errorf("set phases: %w", err)
	}

	// 2) Check if currently enabled
	enabled, err := c.Enabled()
	if err != nil {
		return fmt.Errorf("get enabled state: %w", err)
	}

	// 3) Disable charging to apply new phase setting
	if err := c.Enable(false); err != nil {
		return fmt.Errorf("disable for phase switch: %w", err)
	}

	// 4) Re-enable if it was enabled before
	if enabled {
		if err := c.Enable(true); err != nil {
			return fmt.Errorf("re-enable after phase switch: %w", err)
		}
	}

	return nil
}

// getPhases implements the api.PhaseGetter interface
func (c *HomeAssistant) getPhases() (int, error) {
	if c.phases == "" {
		return 0, errors.New("phase switching not configured")
	}

	// Read the current state of the select entity
	state, err := c.conn.GetStringState(c.phases)
	if err != nil {
		return 0, fmt.Errorf("get phases: %w", err)
	}

	// Parse "1" or "3" from the select state
	phases, err := strconv.Atoi(state)
	if err != nil {
		return 0, fmt.Errorf("invalid phase value %q: %w", state, err)
	}

	// parse phase count from state string
	phases, err := parsePhases(state)
	if err != nil {
		return 0, err
	}

	return phases, nil
}

 // parsePhases extracts the phase count from a select entity state.
 // It accepts:
 //   - bare numeric: "1", "3"
 //   - labeled with leading digit: "1-phase", "3-phase", "1p", "3p"
 //   - labeled with keyword: "single", "three"
 //
 // Returns an error if the state cannot be parsed or is not 1 or 3.
 func parsePhases(state string) (int, error) {
 	state = strings.TrimSpace(state)
 
 	// try direct integer parse first (most common case: "1" or "3")
 	if phases, err := strconv.Atoi(state); err == nil {
 		if phases == 1 || phases == 3 {
 			return phases, nil
 		}
 		return 0, fmt.Errorf("unsupported phase value: %d", phases)
 	}
 
 	// try extracting leading digit (e.g. "1-phase", "3-phase", "1p", "3p")
 	if len(state) > 0 && (state[0] == '1' || state[0] == '3') {
 		return int(state[0] - '0'), nil
 	}
 
 	// try keyword matching (e.g. "single", "three")
 	lower := strings.ToLower(state)
 	if strings.Contains(lower, "single") || strings.Contains(lower, "one") || strings.Contains(lower, "1p") {
 		return 1, nil
 	}
 	if strings.Contains(lower, "three") || strings.Contains(lower, "triple") || strings.Contains(lower, "3p") {
 		return 3, nil
 	}
 
 	return 0, fmt.Errorf("cannot parse phase value from %q: expected '1', '3', or labeled variant", state)
 }

