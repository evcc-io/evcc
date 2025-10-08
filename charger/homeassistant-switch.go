package charger

import (
	"errors"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

type HomeAssistantSwitch struct {
	baseURL      string
	switchEntity string
	powerEntity  string
	conn         *homeassistant.Connection
	*switchSocket
}

func init() {
	registry.Add("homeassistant-switch", NewHomeAssistantSwitchFromConfig)
}

func NewHomeAssistantSwitchFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		embed        `mapstructure:",squash"`
		BaseURL      string
		Token        string
		SwitchEntity string
		PowerEntity  string
		StandbyPower float64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHomeAssistantSwitch(cc.embed, cc.BaseURL, cc.Token, cc.SwitchEntity, cc.PowerEntity, cc.StandbyPower)
}

func NewHomeAssistantSwitch(embed embed, baseURL, token, switchEntity, powerEntity string, standbypower float64) (api.Charger, error) {
	log := util.NewLogger("ha-switch")

	if switchEntity == "" {
		return nil, errors.New("missing switch entity")
	}

	// standbypower < 0 ensures that currentPower is never used by the switch socket if not present
	if powerEntity == "" && standbypower >= 0 {
		return nil, errors.New("missing either power entity or negative standbypower")
	}

	conn, err := homeassistant.NewConnection(log, baseURL, token)
	if err != nil {
		return nil, err
	}

	c := &HomeAssistantSwitch{
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		switchEntity: switchEntity,
		powerEntity:  powerEntity,
		conn:         conn,
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.currentPower, standbypower)

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *HomeAssistantSwitch) Enabled() (bool, error) {
	return c.conn.GetBoolState(c.switchEntity)
}

// Enable implements the api.Charger interface
func (c *HomeAssistantSwitch) Enable(enable bool) error {
	return c.conn.CallSwitchService(c.switchEntity, enable)
}

// currentPower implements the api.Meter interface (optional)
func (c *HomeAssistantSwitch) currentPower() (float64, error) {
	return c.conn.GetFloatState(c.powerEntity)
}
