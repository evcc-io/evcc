package charger

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/homematic"
	"github.com/evcc-io/evcc/util"
)

// Homematic CCU charger implementation
type CCU struct {
	conn *homematic.Connection
	*switchSocket
}

func init() {
	registry.Add("homematic", NewCCUFromConfig)
}

// NewCCUFromConfig creates a Homematic charger from generic config
func NewCCUFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		embed         `mapstructure:",squash"`
		URI           string
		Device        string
		MeterChannel  string
		SwitchChannel string
		User          string
		Password      string
		StandbyPower  float64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewCCU(cc.embed, cc.URI, cc.Device, cc.MeterChannel, cc.SwitchChannel, cc.User, cc.Password, cc.StandbyPower)
}

// NewCCU creates a new connection with standbypower for charger
func NewCCU(embed embed, uri, deviceid, meterid, switchid, user, password string, standbypower float64) (*CCU, error) {
	conn, err := homematic.NewConnection(uri, deviceid, meterid, switchid, user, password)

	c := &CCU{
		conn: conn,
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.conn.CurrentPower, standbypower)

	return c, err
}

// Enabled implements the api.Charger interface
func (c *CCU) Enabled() (bool, error) {
	return c.conn.Enabled()
}

// Enable implements the api.Charger interface
func (c *CCU) Enable(enable bool) error {
	return c.conn.Enable(enable)
}

// MaxCurrent implements the api.Charger interface
func (c *CCU) MaxCurrent(current int64) error {
	return nil
}

var _ api.Meter = (*CCU)(nil)
var _ api.MeterEnergy = (*CCU)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *CCU) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}
