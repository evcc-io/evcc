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
	conn *fritzdect.Connection
	*switchSocket
}

func init() {
	registry.Add("fritzdect", NewFritzDECTFromConfig)
}

// NewFritzDECTFromConfig creates a fritzdect charger from generic config
func NewFritzDECTFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		fritzdect.Settings `mapstructure:",squash"`
		StandbyPower       float64
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewFritzDECT(cc.URI, cc.AIN, cc.User, cc.Password, cc.StandbyPower)
}

// NewFritzDECT creates a new connection with standbypower for charger
func NewFritzDECT(uri, ain, user, password string, standbypower float64) (*FritzDECT, error) {
	conn, err := fritzdect.NewConnection(uri, ain, user, password)

	c := &FritzDECT{
		conn: conn,
	}

	c.switchSocket = NewSwitchSocket(c.Enabled, c.conn.CurrentPower, standbypower)

	return c, err
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

	return c.switchSocket.Status()
}

// Enabled implements the api.Charger interface
func (c *FritzDECT) Enabled() (bool, error) {
	resp, err := c.conn.ExecCmd("getswitchstate")
	if err != nil {
		return false, err
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

var _ api.MeterEnergy = (*FritzDECT)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *FritzDECT) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}
