package charger

//go:generate go tool decorate -f decorateHomeAssistant -b *HomeAssistant -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.CurrentGetter,GetMaxCurrent,func() (float64, error)"

import (
	"errors"
	"fmt"
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
	currentsE  []string
	voltagesE  []string
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
		res, err := homeassistant.ValidatePhaseEntities(cc.Currents)
		if err != nil {
			return nil, fmt.Errorf("currents: %w", err)
		}
		c.currentsE = res
	}

	// Set up phase voltages (optional)
	if len(cc.Voltages) > 0 {
		res, err := homeassistant.ValidatePhaseEntities(cc.Voltages)
		if err != nil {
			return nil, fmt.Errorf("voltages: %w", err)
		}
		c.voltagesE = res
	}

	// decorators for optional interfaces
	var meter func() (float64, error)
	var meterEnergy func() (float64, error)
	var phaseCurrents func() (float64, float64, float64, error)
	var phaseVoltages func() (float64, float64, float64, error)
	var currentGetter func() (float64, error)

	if c.maxcurrent != "" {
		currentGetter = c.getMaxCurrent
	}
	if c.power != "" {
		meter = c.currentPower
	}
	if c.energy != "" {
		meterEnergy = c.totalEnergy
	}
	if c.currentsE != nil {
		phaseCurrents = c.currents
	}
	if c.voltagesE != nil {
		phaseVoltages = c.voltages
	}

	return decorateHomeAssistant(c, meter, meterEnergy, phaseCurrents, phaseVoltages, currentGetter), nil
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

// getMaxCurrent implements the api.CurrentGetter interface
func (c *HomeAssistant) getMaxCurrent() (float64, error) {
	value, err := c.conn.GetFloatState(c.maxcurrent)
	if err != nil {
		return 0, err
	}

	// Return value as integer amperes
	return math.Round(value), nil
}

// currentPower implements the api.Meter interface
func (c *HomeAssistant) currentPower() (float64, error) {
	return c.conn.GetFloatState(c.power)
}

// totalEnergy implements the api.MeterEnergy interface
func (c *HomeAssistant) totalEnergy() (float64, error) {
	return c.conn.GetFloatState(c.energy)
}

// currents implements the api.PhaseCurrents interface
func (c *HomeAssistant) currents() (float64, float64, float64, error) {
	return c.conn.GetPhaseStates(c.currentsE)
}

// voltages implements the api.PhaseVoltages interface
func (c *HomeAssistant) voltages() (float64, float64, float64, error) {
	return c.conn.GetPhaseStates(c.voltagesE)
}
