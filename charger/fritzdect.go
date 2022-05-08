package charger

import (
	"errors"
	"strconv"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/fritzdect"
	"github.com/evcc-io/evcc/util"
)

// AVM FritzBox AHA interface specifications:
// https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AHA-HTTP-Interface.pdf

// FritzDECT charger implementation
type FritzDECT struct {
	conn         *fritzdect.Connection
	standbypower float64
}

func init() {
	registry.Add("fritzdect", NewFritzDECTFromConfig)
}

// NewFritzDECTFromConfig creates a fritzdect charger from generic config
func NewFritzDECTFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI          string
		AIN          string
		User         string
		Password     string
		StandbyPower float64
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewFritzDECT(cc.URI, cc.AIN, cc.User, cc.Password, cc.StandbyPower)
}

// NewFritzDECT creates a new connection with standbypower for charger
func NewFritzDECT(uri, ain, user, password string, standbypower float64) (*FritzDECT, error) {
	conn, err := fritzdect.NewConnection(uri, ain, user, password)

	fd := &FritzDECT{
		conn:         conn,
		standbypower: standbypower,
	}

	return fd, err
}

// Enabled implements the api.Charger interface
func (c *FritzDECT) Enabled() (bool, error) {
	resp, err := c.conn.ExecCmd("getswitchstate")
	if err != nil {
		return false, err
	}

	if resp == "inval" {
		return false, api.ErrNotAvailable
	}

	return strconv.ParseBool(resp)
}

// Enable implements the api.Charger interface
func (c *FritzDECT) Enable(enable bool) error {
	cmd := "setswitchoff"
	if enable {
		cmd = "setswitchon"
	}

	// on 0/1 - DECT Switch state off/on (empty if unknown or error)
	resp, err := c.conn.ExecCmd(cmd)

	var on bool
	if err == nil {
		on, err = strconv.ParseBool(resp)
		if err == nil && enable != on {
			err = errors.New("switch failed")
		}
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *FritzDECT) MaxCurrent(current int64) error {
	return nil
}

// Status implements the api.Charger interface
func (c *FritzDECT) Status() (api.ChargeStatus, error) {
	resp, err := c.conn.ExecCmd("getswitchpresent")

	if err == nil {
		var present bool
		present, err = strconv.ParseBool(resp)
		if err == nil && !present {
			err = api.ErrNotAvailable
		}
	}
	if err != nil {
		return api.StatusNone, err
	}

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

var _ api.Meter = (*FritzDECT)(nil)

// CurrentPower implements the api.Meter interface
func (c *FritzDECT) CurrentPower() (float64, error) {
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

var _ api.MeterEnergy = (*FritzDECT)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *FritzDECT) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}
