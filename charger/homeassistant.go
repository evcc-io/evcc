package charger

//go:generate go tool decorate -f decorateHomeAssistant -b *HomeAssistant -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)"
//  -t "api.CurrentGetter,GetMaxCurrent,func() (float64, error)"

import (
	"errors"
	"fmt"

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
	cc := struct {
		URI        string
		Token      string
		Status     string   // required - sensor for charge status
		Enabled    string   // required - sensor for enabled state
		Enable     string   // required - switch/input_boolean for enable/disable
		MaxCurrent string   // required - number entity for setting max current
		Power      string   // optional - power sensor
		Energy     string   // optional - energy sensor
		Currents   []string // optional - current sensors for L1, L2, L3
		Voltages   []string // optional - voltage sensors for L1, L2, L3
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

	log := util.NewLogger("ha-charger")
	conn, err := homeassistant.NewConnection(log, cc.URI, cc.Token)
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

	return decorateHomeAssistant(c, power, energy, currents, voltages), nil
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
	return c.conn.CallNumberService(c.maxcurrent, float64(current))
}
