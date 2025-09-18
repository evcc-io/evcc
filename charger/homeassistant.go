package charger

import (
	"errors"
	"math"

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
	power      string
	energy     string
	currents   [3]string
	voltages   [3]string
	maxcurrent string
}

func init() {
	registry.Add("homeassistant", NewHomeAssistantFromConfig)
}

// NewHomeAssistantFromConfig creates a HomeAssistant charger from generic config
func NewHomeAssistantFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		BaseURL    string
		Token      string
		Status     string   // required - sensor for charge status
		Enabled    string   // required - sensor for enabled state
		Enable     string   // required - switch/input_boolean for enable/disable
		Power      string   // optional - power sensor
		Energy     string   // optional - energy sensor
		Currents   []string // optional - current sensors for L1, L2, L3
		Voltages   []string // optional - voltage sensors for L1, L2, L3
		MaxCurrent string   // optional - number entity for setting max current
	}{}

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

	conn, err := homeassistant.NewConnection(cc.BaseURL, cc.Token)
	if err != nil {
		return nil, err
	}

	c := &HomeAssistant{
		conn:       conn,
		status:     cc.Status,
		enabled:    cc.Enabled,
		enable:     cc.Enable,
		power:      cc.Power,
		energy:     cc.Energy,
		maxcurrent: cc.MaxCurrent,
	}

	// Set up phase currents (optional)
	if len(cc.Currents) > 0 {
		currents, err := homeassistant.ValidatePhaseEntities(cc.Currents, "currents")
		if err != nil {
			return nil, err
		}
		c.currents = currents
	}

	// Set up phase voltages (optional)
	if len(cc.Voltages) > 0 {
		voltages, err := homeassistant.ValidatePhaseEntities(cc.Voltages, "voltages")
		if err != nil {
			return nil, err
		}
		c.voltages = voltages
	}

	return c, nil
}

var _ api.Charger = (*HomeAssistant)(nil)

// Status implements the api.ChargeState interface
func (c *HomeAssistant) Status() (api.ChargeStatus, error) {
	state, err := c.conn.GetState(c.status)
	if err != nil {
		return api.StatusNone, err
	}

	return homeassistant.ParseChargeStatus(state)
}

// Enabled implements the api.Charger interface
func (c *HomeAssistant) Enabled() (bool, error) {
	return c.conn.GetBoolState(c.enabled)
}

// Enable implements the api.Charger interface
func (c *HomeAssistant) Enable(enable bool) error {
	return c.conn.CallSwitchService(c.enable, enable)
}

// MaxCurrent implements the api.CurrentController interface
func (c *HomeAssistant) MaxCurrent(current int64) error {
	if c.maxcurrent == "" {
		return api.ErrNotAvailable
	}

	return c.conn.CallNumberService(c.maxcurrent, float64(current))
}

var _ api.Meter = (*HomeAssistant)(nil)

// CurrentPower implements the api.Meter interface
func (c *HomeAssistant) CurrentPower() (float64, error) {
	if c.power == "" {
		return 0, api.ErrNotAvailable
	}
	return c.conn.GetFloatState(c.power)
}

var _ api.MeterEnergy = (*HomeAssistant)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *HomeAssistant) TotalEnergy() (float64, error) {
	if c.energy == "" {
		return 0, api.ErrNotAvailable
	}
	return c.conn.GetFloatState(c.energy)
}

var _ api.PhaseCurrents = (*HomeAssistant)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *HomeAssistant) Currents() (float64, float64, float64, error) {
	if c.currents[0] == "" {
		return 0, 0, 0, api.ErrNotAvailable
	}
	return c.conn.GetPhaseStates(c.currents)
}

var _ api.CurrentGetter = (*HomeAssistant)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (c *HomeAssistant) GetMaxCurrent() (float64, error) {
	if c.maxcurrent == "" {
		return 0, api.ErrNotAvailable
	}

	value, err := c.conn.GetFloatState(c.maxcurrent)
	if err != nil {
		return 0, err
	}

	// Return value as integer amperes
	return math.Round(value), nil
}

var _ api.PhaseVoltages = (*HomeAssistant)(nil)

// Voltages implements the api.PhaseVoltages interface
func (c *HomeAssistant) Voltages() (float64, float64, float64, error) {
	if c.voltages[0] == "" {
		return 0, 0, 0, api.ErrNotAvailable
	}
	return c.conn.GetPhaseStates(c.voltages)
}