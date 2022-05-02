package charger

import (
	"errors"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/tapo"
	"github.com/evcc-io/evcc/util"
)

// TP-Link Tapo charger implementation
type Tapo struct {
	conn         *tapo.Connection
	standbypower float64
}

func init() {
	registry.Add("tapo", NewTapoFromConfig)
}

// NewTapoFromConfig creates a Tapo charger from generic config
func NewTapoFromConfig(other map[string]interface{}) (api.Charger, error) {
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

	return NewTapo(cc.URI, cc.User, cc.Password, cc.StandbyPower)
}

// NewTapo creates Tapo charger
func NewTapo(uri, user, password string, standbypower float64) (*Tapo, error) {
	for _, suffix := range []string{"/", "/app"} {
		uri = strings.TrimSuffix(uri, suffix)
	}

	conn, err := tapo.NewConnection(uri, user, password)
	if err != nil {
		return nil, err
	}

	tapo := &Tapo{
		conn:         conn,
		standbypower: standbypower,
	}

	if user == "" || password == "" {
		return tapo, fmt.Errorf("missing user or password")
	}

	return tapo, nil
}

// Enabled implements the api.Charger interface
func (c *Tapo) Enabled() (bool, error) {
	resp, err := c.conn.ExecTapoCmd("get_device_info", false)
	if err != nil {
		return false, err
	}
	return resp.Result.DeviceON, nil
}

// Enable implements the api.Charger interface
func (c *Tapo) Enable(enable bool) error {
	_, err := c.conn.ExecTapoCmd("set_device_info", enable)
	return err
}

// MaxCurrent implements the api.Charger interface
func (c *Tapo) MaxCurrent(current int64) error {
	return nil
}

// Status implements the api.Charger interface
func (c *Tapo) Status() (api.ChargeStatus, error) {
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

var _ api.Meter = (*Tapo)(nil)

// CurrentPower implements the api.Meter interface
func (c *Tapo) CurrentPower() (float64, error) {
	var power float64

	// set fix static power in static mode
	if c.standbypower < 0 {
		on, err := c.Enabled()
		if err != nil {
			return 0, err
		}
		if on {
			power = -c.standbypower
		}
		return power, nil
	}

	// standby power mode
	power, err := c.conn.CurrentPower()
	if err != nil {
		return 0, err
	}

	// ignore power in standby mode
	if c.standbypower >= 0 && power <= c.standbypower {
		power = 0
	}

	return power, nil
}

var _ api.ChargeRater = (*Tapo)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *Tapo) ChargedEnergy() (float64, error) {
	return c.conn.ChargedEnergy()
}
