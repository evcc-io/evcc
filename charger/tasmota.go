package charger

import (
	"errors"
	"fmt"
	"strings"
	"time"

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
		embed        `mapstructure:",squash"`
		URI          string
		User         string
		Password     string
		StandbyPower float64
		Channel      int
		Cache        time.Duration
	}{
		Channel: 1,
		Cache:   time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewTasmota(cc.embed, cc.URI, cc.User, cc.Password, cc.Channel, cc.StandbyPower, cc.Cache)
}

// NewTasmota creates Tasmota charger
func NewTasmota(embed embed, uri, user, password string, channel int, standbypower float64, cache time.Duration) (*Tasmota, error) {
	conn, err := tasmota.NewConnection(uri, user, password, channel, cache)
	if err != nil {
		return nil, err
	}

	c := &Tasmota{
		conn:    conn,
		channel: channel,
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.conn.CurrentPower, standbypower)

	return c, c.conn.ChannelExists(channel)
}

// Enabled implements the api.Charger interface
func (c *Tasmota) Enabled() (bool, error) {
	return c.conn.Enabled()
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

var _ api.MeterEnergy = (*Tasmota)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Tasmota) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}
