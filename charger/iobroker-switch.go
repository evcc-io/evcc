package charger

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/iobroker"
)

type IobrokerSwitch struct {
	conn   *iobroker.Connection
	enable string
	power  string
	*switchSocket
}

func init() {
	registry.Add("iobroker-switch", NewIobrokerSwitchFromConfig)
}

func NewIobrokerSwitchFromConfig(other map[string]any) (api.Charger, error) {
	var cc struct {
		embed        `mapstructure:",squash"`
		Name         string `mapstructure:"iobroker"`
		Enable       string
		Power        string
		StandbyPower float64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewIobrokerSwitch(cc.embed, cc.Name, cc.Enable, cc.Power, cc.StandbyPower)
}

func NewIobrokerSwitch(embed embed, name, enable, power string, standbypower float64) (api.Charger, error) {
	if enable == "" {
		return nil, errors.New("missing enable switch entity")
	}

	// standbypower < 0 ensures that currentPower is never used by the switch socket if not present
	if power == "" && standbypower >= 0 {
		return nil, errors.New("missing either power entity or negative standbypower")
	}

	conn := iobroker.GetConnection(name)
	if conn == nil {
		return nil, errors.New("referenced iobroker instance unknown")
	}

	c := &IobrokerSwitch{
		enable: enable,
		power:  power,
		conn:   conn,
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.currentPower, standbypower)

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *IobrokerSwitch) Enabled() (bool, error) {
	return c.conn.GetBoolState(c.enable)
}

// Enable implements the api.Charger interface
func (c *IobrokerSwitch) Enable(enable bool) error {
	return c.conn.SetBoolState(c.enable, enable)
}

// currentPower implements the api.Meter interface (optional)
func (c *IobrokerSwitch) currentPower() (float64, error) {
	return c.conn.GetFloatState(c.power)
}
