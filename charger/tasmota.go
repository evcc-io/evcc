package charger

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/tasmota"
	"github.com/evcc-io/evcc/util"
)

// Tasmota project homepage
// https://tasmota.github.io/docs/
// Supported devices:
// https://templates.blakadder.com/

// Tasmota charger implementation
type Tasmota struct {
	conn         *tasmota.Connection
	standbypower float64
}

func init() {
	registry.Add("tasmota", NewTasmotaFromConfig)
}

// NewTasmotaFromConfig creates a Tasmota charger from generic config
func NewTasmotaFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI          string
		User         string
		Password     string
		StandbyPower float64
	}{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return NewTasmota(cc.URI, cc.User, cc.Password, cc.StandbyPower)
}

// NewTasmota creates Tasmota charger
func NewTasmota(uri, user, password string, standbypower float64) (*Tasmota, error) {
	conn, err := tasmota.NewConnection(uri, user, password)
	if err != nil {
		return nil, err
	}

	c := &Tasmota{
		conn:         conn,
		standbypower: standbypower,
	}

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *Tasmota) Enabled() (bool, error) {
	var res tasmota.StatusResponse
	err := c.conn.GetJSON(c.conn.CreateCmd("Status 0"), &res)

	return res.Status.Power == 1, err
}

// Enable implements the api.Charger interface
func (c *Tasmota) Enable(enable bool) error {
	var res tasmota.PowerResponse
	cmd := "Power off"
	if enable {
		cmd = "Power on"
	}
	err := c.conn.GetJSON(c.conn.CreateCmd(cmd), &res)

	switch {
	case err != nil:
		return err
	case enable && res.Power != "ON":
		return errors.New("switchOn failed")
	case !enable && res.Power != "OFF":
		return errors.New("switchOff failed")
	default:
		return nil
	}
}

// MaxCurrent implements the api.Charger interface
func (c *Tasmota) MaxCurrent(current int64) error {
	return nil
}

// Status implements the api.Charger interface
func (c *Tasmota) Status() (api.ChargeStatus, error) {
	res := api.StatusB

	// static mode
	if c.standbypower < 0 {
		on, err := c.Enabled()
		if on {
			res = api.StatusC
		}

		return res, err
	}

	// standby power mode
	power, err := c.CurrentPower()
	if power > c.standbypower {
		res = api.StatusC
	}

	return res, err
}

var _ api.Meter = (*Tasmota)(nil)

// CurrentPower implements the api.Meter interface
func (c *Tasmota) CurrentPower() (float64, error) {
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
	if c.standbypower >= 0 && power <= c.standbypower {
		power = 0
	}

	return power, err
}

var _ api.MeterEnergy = (*Tasmota)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Tasmota) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}
