package charger

import (
	"time"

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
	cc := struct {
		embed         `mapstructure:",squash"`
		URI           string
		Device        string
		MeterChannel  string
		SwitchChannel string
		User          string
		Password      string
		StandbyPower  float64
		Cache         time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewCCU(cc.embed, cc.URI, cc.Device, cc.MeterChannel, cc.SwitchChannel, cc.User, cc.Password, cc.StandbyPower, cc.Cache)
}

// NewCCU creates a new connection with standbypower for charger
func NewCCU(embed embed, uri, deviceid, meterid, switchid, user, password string, standbypower float64, cache time.Duration) (*CCU, error) {
	conn, err := homematic.NewConnection(uri, deviceid, meterid, switchid, user, password, cache)

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

var _ api.MeterEnergy = (*CCU)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *CCU) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}
