package charger

import (
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
	return c.conn.Enable(enable)
}

var _ api.MeterEnergy = (*Tasmota)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Tasmota) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}
