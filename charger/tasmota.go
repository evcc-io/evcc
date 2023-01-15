package charger

import (
	"errors"
	"fmt"
	"strings"

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
	conn    *tasmota.Connection
	channel int
	*switchSocket
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
		Channel      int
	}{
		Channel: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewTasmota(cc.URI, cc.User, cc.Password, cc.Channel, cc.StandbyPower)
}

// NewTasmota creates Tasmota charger
func NewTasmota(uri, user, password string, channel int, standbypower float64) (*Tasmota, error) {
	conn, err := tasmota.NewConnection(uri, user, password, channel)
	if err != nil {
		return nil, err
	}

	c := &Tasmota{
		conn:    conn,
		channel: channel,
	}

	c.switchSocket = NewSwitchSocket(c.Enabled, c.conn.CurrentPower, standbypower)

	return c, c.channelExists(channel)
}

// channelExists checks the existence of the configured relay channel interface
func (c *Tasmota) channelExists(channel int) error {
	var res *tasmota.StatusSTSResponse
	if err := c.conn.ExecCmd("Status 0", &res); err != nil {
		return err
	}

	var ok bool
	switch channel {
	case 1:
		ok = res.StatusSTS.Power != "" || res.StatusSTS.Power1 != ""
	case 2:
		ok = res.StatusSTS.Power2 != ""
	case 3:
		ok = res.StatusSTS.Power3 != ""
	case 4:
		ok = res.StatusSTS.Power4 != ""
	case 5:
		ok = res.StatusSTS.Power5 != ""
	case 6:
		ok = res.StatusSTS.Power6 != ""
	case 7:
		ok = res.StatusSTS.Power7 != ""
	case 8:
		ok = res.StatusSTS.Power8 != ""
	}

	if !ok {
		return fmt.Errorf("invalid relay channel: %d", channel)
	}

	return nil
}

// Enabled implements the api.Charger interface
func (c *Tasmota) Enabled() (bool, error) {
	var res tasmota.StatusSTSResponse
	err := c.conn.ExecCmd("Status 0", &res)
	if err != nil {
		return false, err
	}

	switch c.channel {
	case 2:
		return strings.ToUpper(res.StatusSTS.Power2) == "ON", err
	case 3:
		return strings.ToUpper(res.StatusSTS.Power3) == "ON", err
	case 4:
		return strings.ToUpper(res.StatusSTS.Power4) == "ON", err
	case 5:
		return strings.ToUpper(res.StatusSTS.Power5) == "ON", err
	case 6:
		return strings.ToUpper(res.StatusSTS.Power6) == "ON", err
	case 7:
		return strings.ToUpper(res.StatusSTS.Power7) == "ON", err
	case 8:
		return strings.ToUpper(res.StatusSTS.Power8) == "ON", err
	default:
		return strings.ToUpper(res.StatusSTS.Power) == "ON" || strings.ToUpper(res.StatusSTS.Power1) == "ON", err
	}
}

// Enable implements the api.Charger interface
func (c *Tasmota) Enable(enable bool) error {
	var res tasmota.PowerResponse

	cmd := fmt.Sprintf("Power%d off", c.channel)
	if enable {
		cmd = fmt.Sprintf("Power%d on", c.channel)
	}

	if err := c.conn.ExecCmd(cmd, &res); err != nil {
		return err
	}

	var on bool
	switch c.channel {
	case 2:
		on = strings.ToUpper(res.Power2) == "ON"
	case 3:
		on = strings.ToUpper(res.Power3) == "ON"
	case 4:
		on = strings.ToUpper(res.Power4) == "ON"
	case 5:
		on = strings.ToUpper(res.Power5) == "ON"
	case 6:
		on = strings.ToUpper(res.Power6) == "ON"
	case 7:
		on = strings.ToUpper(res.Power7) == "ON"
	case 8:
		on = strings.ToUpper(res.Power8) == "ON"
	default:
		on = strings.ToUpper(res.Power) == "ON" || strings.ToUpper(res.Power1) == "ON"
	}

	switch {
	case enable && !on:
		return errors.New("switchOn failed")
	case !enable && on:
		return errors.New("switchOff failed")
	default:
		return nil
	}
}

// MaxCurrent implements the api.Charger interface
func (c *Tasmota) MaxCurrent(current int64) error {
	return nil
}

var _ api.MeterEnergy = (*Tasmota)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Tasmota) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}
