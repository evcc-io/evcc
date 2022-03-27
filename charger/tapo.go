package charger

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/tapo"
	"github.com/evcc-io/evcc/util"
)

// TP-Link Tapo charger implementation
type Tapo struct {
	conn            *tapo.Connection
	standbypower    float64
	updated         time.Time
	lasttodayenergy int64
	energy          int64
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

	conn := tapo.NewConnection(uri, user, password)

	tapo := &Tapo{
		conn:         conn,
		standbypower: standbypower,
	}

	if user == "" || password == "" {
		return tapo, fmt.Errorf("missing user/password")
	}

	return tapo, nil
}

// Enabled implements the api.Charger interface
func (c *Tapo) Enabled() (bool, error) {
	resp, err := c.execTapoCmd("get_device_info", false)
	if err != nil {
		return false, err
	}
	return resp.Result.DeviceON, nil
}

// Enable implements the api.Charger interface
func (c *Tapo) Enable(enable bool) error {
	_, err := c.execTapoCmd("set_device_info", enable)
	return err
}

// MaxCurrent implements the api.Charger interface
func (c *Tapo) MaxCurrent(current int64) error {
	return nil
}

// Status implements the api.Charger interface
func (c *Tapo) Status() (api.ChargeStatus, error) {
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

var _ api.Meter = (*Tapo)(nil)

// CurrentPower implements the api.Meter interface
func (c *Tapo) CurrentPower() (float64, error) {
	resp, err := c.execTapoCmd("get_energy_usage", false)
	if err != nil {
		return 0, err
	}
	return float64(resp.Result.Current_Power) / 1000, nil
}

var _ api.ChargeRater = (*Vestel)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *Tapo) ChargedEnergy() (float64, error) {
	resp, err := c.execTapoCmd("get_energy_usage", false)
	if err != nil {
		return 0, err
	}

	if resp.Result.Today_Energy > c.lasttodayenergy {
		c.energy = c.energy + (resp.Result.Today_Energy - c.lasttodayenergy)
	}
	c.lasttodayenergy = resp.Result.Today_Energy

	return float64(c.energy) / 1000, nil
}

// execTapoCmd executes a Tapo api command and provides the response
func (c *Tapo) execTapoCmd(method string, enable bool) (*tapo.DeviceResponse, error) {
	// refresh Tapo session id
	if time.Since(c.updated) >= 600*time.Minute {
		err := c.conn.Login()
		if err != nil {
			return nil, err
		}
		// update session timestamp
		c.updated = time.Now()
	}

	return c.conn.ExecMethod(method, enable)
}
