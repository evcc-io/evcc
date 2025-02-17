package charger

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/shelly"
	"github.com/evcc-io/evcc/util"
)

// Shelly charger implementation
type Shelly struct {
	conn *shelly.Switch
	*switchSocket
}

func init() {
	registry.Add("shelly", NewShellyFromConfig)
}

// NewShellyFromConfig creates a Shelly charger from generic config
func NewShellyFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		embed        `mapstructure:",squash"`
		URI          string
		User         string
		Password     string
		Channel      int
		StandbyPower float64
		Invert       bool
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewShelly(cc.embed, cc.URI, cc.User, cc.Password, cc.Channel, cc.StandbyPower, cc.Invert)
}

// NewShelly creates Shelly charger
func NewShelly(embed embed, uri, user, password string, channel int, standbypower float64, invert bool) (*Shelly, error) {
	conn, err := shelly.NewConnection(uri, user, password, channel)
	if err != nil {
		return nil, err
	}

	c := &Shelly{
		conn: shelly.NewSwitch(conn, "charge", invert),
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.conn.CurrentPower, standbypower)

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *Shelly) Enabled() (bool, error) {
	return c.conn.Enabled()
}

// Enable implements the api.Charger interface
func (c *Shelly) Enable(enable bool) error {
	if err := c.conn.Enable(enable); err != nil {
		return err
	}

	enabled, err := c.Enabled()
	switch {
	case err != nil:
		return err
	case enable != enabled:
		onoff := map[bool]string{true: "on", false: "off"}
		return fmt.Errorf("switch %s failed", onoff[enable])
	default:
		return nil
	}
}

var _ api.MeterEnergy = (*Shelly)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Shelly) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}
