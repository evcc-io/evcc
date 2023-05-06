package charger

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/tapo"
	"github.com/evcc-io/evcc/util"
)

// TP-Link Tapo charger implementation
type Tapo struct {
	conn *tapo.Connection
	*switchSocket
}

func init() {
	registry.Add("tapo", NewTapoFromConfig)
}

// NewTapoFromConfig creates a Tapo charger from generic config
func NewTapoFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		embed        `mapstructure:",squash"`
		URI          string
		User         string
		Password     string
		StandbyPower float64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewTapo(cc.embed, cc.URI, cc.User, cc.Password, cc.StandbyPower)
}

// NewTapo creates Tapo charger
func NewTapo(embed embed, uri, user, password string, standbypower float64) (*Tapo, error) {
	conn, err := tapo.NewConnection(uri, user, password)
	if err != nil {
		return nil, err
	}

	c := &Tapo{
		conn: conn,
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.conn.CurrentPower, standbypower)

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *Tapo) Enabled() (bool, error) {
	resp, err := c.conn.ExecCmd("get_device_info", false)
	if err != nil {
		return false, err
	}
	return resp.Result.DeviceON, nil
}

// Enable implements the api.Charger interface
func (c *Tapo) Enable(enable bool) error {
	_, err := c.conn.ExecCmd("set_device_info", enable)
	return err
}

var _ api.ChargeRater = (*Tapo)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *Tapo) ChargedEnergy() (float64, error) {
	return c.conn.ChargedEnergy()
}
