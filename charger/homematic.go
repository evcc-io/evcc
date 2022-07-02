package charger

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/homematic"
	"github.com/evcc-io/evcc/util"
)

// Homematic CCU charger implementation
type CCU struct {
	conn         *homematic.Connection
	standbypower float64
}

func init() {
	registry.Add("homematic", NewCCUFromConfig)
}

// NewCCUFromConfig creates a fritzdect charger from generic config
func NewCCUFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI          string
		DeviceId     string
		MeterId      string
		SwitchId     string
		User         string
		Password     string
		StandbyPower float64
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewCCU(cc.URI, cc.DeviceId, cc.MeterId, cc.SwitchId, cc.User, cc.Password, cc.StandbyPower)
}

// NewCCU creates a new connection with standbypower for charger
func NewCCU(uri, deviceid, meterid, switchid, user, password string, standbypower float64) (*CCU, error) {
	conn := homematic.NewConnection(uri, deviceid, meterid, switchid, user, password)

	fd := &CCU{
		conn:         conn,
		standbypower: standbypower,
	}

	return fd, nil
}

// Enabled implements the api.Charger interface
func (c *CCU) Enabled() (bool, error) {
	return c.conn.Enabled()
}

// Enable implements the api.Charger interface
func (c *CCU) Enable(enable bool) error {
	return nil
}

// MaxCurrent implements the api.Charger interface
func (c *CCU) MaxCurrent(current int64) error {
	return nil
}

// Status implements the api.Charger interface
func (c *CCU) Status() (api.ChargeStatus, error) {
	res := api.StatusB
	on, err := c.Enabled()
	if err != nil {
		return res, err
	}

	power, err := c.conn.CurrentPower()
	if err != nil {
		return res, err
	}

	// static mode || standby power mode condition
	if on && (c.standbypower < 0 || power > c.standbypower) {
		res = api.StatusC
	}

	return res, nil
}

var _ api.Meter = (*CCU)(nil)

// CurrentPower implements the api.Meter interface
func (c *CCU) CurrentPower() (float64, error) {
	var power float64

	// set fix static power in static mode
	if c.standbypower < 0 {
		on, err := c.Enabled()
		if on {
			power = -c.standbypower
		}
		return power, err
	}

	// ignore power in standby mode
	power, err := c.conn.CurrentPower()
	if power <= c.standbypower {
		power = 0
	}

	return power, err
}

var _ api.MeterEnergy = (*CCU)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *CCU) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}
