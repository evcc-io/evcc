package charger

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

type HomeAssistantSwitch struct {
	conn   *homeassistant.Connection
	enable string
	power  string
	*switchSocket
}

func init() {
	registry.Add("homeassistant-switch", NewHomeAssistantSwitchFromConfig)
}

func NewHomeAssistantSwitchFromConfig(other map[string]any) (api.Charger, error) {
	var cc struct {
		embed        `mapstructure:",squash"`
		URI          string
		Token_       string `mapstructure:"token"` // TODO deprecated
		Home         string // TODO deprecated
		Enable       string
		Power        string
		StandbyPower float64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHomeAssistantSwitch(cc.embed, cc.URI, cc.Home, cc.Enable, cc.Power, cc.StandbyPower)
}

func NewHomeAssistantSwitch(embed embed, uri, home, enable, power string, standbypower float64) (api.Charger, error) {
	if enable == "" {
		return nil, errors.New("missing enable switch entity")
	}

	// standbypower < 0 ensures that currentPower is never used by the switch socket if not present
	if power == "" && standbypower >= 0 {
		return nil, errors.New("missing either power entity or negative standbypower")
	}

	log := util.NewLogger("ha-switch")

	conn, err := homeassistant.NewConnection(log, uri, home)
	if err != nil {
		return nil, err
	}

	c := &HomeAssistantSwitch{
		enable: enable,
		power:  power,
		conn:   conn,
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.currentPower, standbypower)

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *HomeAssistantSwitch) Enabled() (bool, error) {
	return c.conn.GetBoolState(c.enable)
}

// Enable implements the api.Charger interface
func (c *HomeAssistantSwitch) Enable(enable bool) error {
	return c.conn.CallSwitchService(c.enable, enable)
}

// currentPower implements the api.Meter interface (optional)
func (c *HomeAssistantSwitch) currentPower() (float64, error) {
	return c.conn.GetFloatState(c.power)
}
