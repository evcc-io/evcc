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
	conn            *homeassistant.Connection
	status          string
	enabled         string
	enable          string
	power           string
	energy          string
	currentEntities [3]string
	voltageEntities [3]string
	maxcurrent      string
}

// parsePhases helper to turn a []string into a [3]string or error
func parsePhases(name string, cfg []string) ([3]string, error) {
	var arr [3]string
	if len(cfg) == 0 {
		return arr, nil
	}
	if len(cfg) != 1 && len(cfg) != 3 {
		return arr, fmt.Errorf("%s must contain either 1 entity (single-phase) or 3 entities (three-phase L1, L2, L3), got %d", name, len(cfg))
	}
	if len(cfg) == 1 {
		arr[0] = cfg[0]
	} else {
		copy(arr[:], cfg)
	}
	return arr, nil
}

func init() {
	registry.Add("homeassistant", NewHomeAssistantFromConfig)
}

// NewHomeAssistantFromConfig creates a HomeAssistant charger from generic config
func NewHomeAssistantFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI        string   `mapstructure:"uri"`
		Token      string   `mapstructure:"token"`
		Status     string   `mapstructure:"status"`     // required - sensor for charge status
		Enabled    string   `mapstructure:"enabled"`    // required - sensor for enabled state
		Enable     string   `mapstructure:"enable"`     // required - switch/input_boolean for enable/disable
		Power      string   `mapstructure:"power"`      // optional - power sensor
		Energy     string   `mapstructure:"energy"`     // optional - energy sensor
		Currents   []string `mapstructure:"currents"`   // optional - current sensors for L1, L2, L3
		Voltages   []string `mapstructure:"voltages"`   // optional - voltage sensors for L1, L2, L3
		MaxCurrent string   `mapstructure:"maxcurrent"` // optional - number entity for setting max current
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

	conn, err := homeassistant.NewConnection(cc.URI, cc.Token)
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
	currents, err := parsePhases("currents", cc.Currents)
	if err != nil {
		return nil, err
	}
	c.currentEntities = currents

	// Set up phase voltages (optional)
	voltages, err := parsePhases("voltages", cc.Voltages)
	if err != nil {
		return nil, err
	}
	c.voltageEntities = voltages

	// decorators for optional interfaces
	var meter func() (float64, error)
	var meterEnergy func() (float64, error)
	var phaseCurrents func() (float64, float64, float64, error)
	var phaseVoltages func() (float64, float64, float64, error)
	var currentGetter func() (float64, error)

	if c.power != "" {
		meter = c.currentPower
	}
	if c.energy != "" {
		meterEnergy = c.totalEnergy
	}
	if c.currentEntities[0] != "" {
		phaseCurrents = c.currents
	}
	if c.voltageEntities[0] != "" {
		phaseVoltages = c.voltages
	}
	if c.maxcurrent != "" {
		currentGetter = c.getMaxCurrent
	}

	return decorateHomeAssistant(c, meter, meterEnergy, phaseCurrents, phaseVoltages, currentGetter), nil
}

// Helper function to reduce duplication for optional interfaces
func (c *HomeAssistant) optFloat(entity string) (float64, error) {
	if entity == "" {
		return 0, api.ErrNotAvailable
	}
	return c.conn.GetFloatState(entity)
}

func (c *HomeAssistant) optCallNumber(entity string, value float64) error {
	if entity == "" {
		return api.ErrNotAvailable
	}
	return c.conn.CallNumberService(entity, value)
}

func (c *HomeAssistant) optPhaseCurrents() (float64, float64, float64, error) {
	if c.currentEntities[0] == "" {
		return 0, 0, 0, api.ErrNotAvailable
	}
	return c.conn.GetPhaseStates(c.currentEntities)
}

func (c *HomeAssistant) optPhaseVoltages() (float64, float64, float64, error) {
	if c.voltageEntities[0] == "" {
		return 0, 0, 0, api.ErrNotAvailable
	}
	return c.conn.GetPhaseStates(c.voltageEntities)
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
	return c.optCallNumber(c.maxcurrent, float64(current))
}

// currentPower implements the api.Meter interface (private for decorator)
func (c *HomeAssistant) currentPower() (float64, error) {
	return c.optFloat(c.power)
}

// totalEnergy implements the api.MeterEnergy interface (private for decorator)
func (c *HomeAssistant) totalEnergy() (float64, error) {
	return c.optFloat(c.energy)
}

// currents implements the api.PhaseCurrents interface (private for decorator)
func (c *HomeAssistant) currents() (float64, float64, float64, error) {
	return c.optPhaseCurrents()
}

// getMaxCurrent implements the api.CurrentGetter interface (private for decorator)
func (c *HomeAssistant) getMaxCurrent() (float64, error) {
	value, err := c.optFloat(c.maxcurrent)
	if err != nil {
		return 0, err
	}

	// Return value as integer amperes
	return math.Round(value), nil
}

// voltages implements the api.PhaseVoltages interface (private for decorator)
func (c *HomeAssistant) voltages() (float64, float64, float64, error) {
	return c.optPhaseVoltages()
}
