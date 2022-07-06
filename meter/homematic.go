package meter

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/homematic"
	"github.com/evcc-io/evcc/util"
)

// Homematic CCU charger implementation
type CCU struct {
	conn  *homematic.Connection
	usage string
}

func init() {
	registry.Add("homematic", NewCCUFromConfig)
}

// NewCCUFromConfig creates a homematic meter from generic config
func NewCCUFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI           string
		Device        string
		MeterChannel  string
		SwitchChannel string
		User          string
		Password      string
		Usage         string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewCCU(cc.URI, cc.Device, cc.MeterChannel, cc.SwitchChannel, cc.User, cc.Password, cc.Usage)
}

// NewCCU creates a new connection with usage for meter
func NewCCU(uri, deviceid, meterid, switchid, user, password, usage string) (*CCU, error) {
	conn, err := homematic.NewConnection(uri, deviceid, meterid, switchid, user, password)
	if err != nil {
		return nil, err
	}

	fd := &CCU{
		conn:  conn,
		usage: usage,
	}

	return fd, nil
}

var _ api.Meter = (*CCU)(nil)

// CurrentPower implements the api.Meter interface
func (c *CCU) CurrentPower() (float64, error) {
	if c.usage == "grid" {
		return c.conn.GridCurrentPower()
	}
	return c.conn.CurrentPower()
}

var _ api.MeterEnergy = (*CCU)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *CCU) TotalEnergy() (float64, error) {
	if c.usage == "grid" {
		return c.conn.GridTotalEnergy()
	}
	return c.conn.TotalEnergy()
}
