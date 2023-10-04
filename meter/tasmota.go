package meter

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/tasmota"
	"github.com/evcc-io/evcc/util"
)

// Tasmota meter implementation
type Tasmota struct {
	conn  *tasmota.Connection
	usage string
}

// Tasmota meter implementation
func init() {
	registry.Add("tasmota", NewTasmotaFromConfig)
}

// NewTasmotaFromConfig creates a Tasmota meter from generic config
func NewTasmotaFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI      string
		User     string
		Password string
		Channel  int
		Channels []int
		Usage    string
		Cache    time.Duration
	}{
		Channels: []int{1},
		Cache:    time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// Only for backward compatibility, for users having deprecated single channel config
	if cc.Channel != 0 {
		cc.Channels = []int{1}
		cc.Channels[0] = cc.Channel
	}

	return NewTasmota(cc.URI, cc.User, cc.Password, cc.Usage, cc.Channels, cc.Cache)
}

// NewTasmota creates Tasmota meter
func NewTasmota(uri, user, password, usage string, channels []int, cache time.Duration) (*Tasmota, error) {
	conn, err := tasmota.NewConnection(uri, user, password, channels, cache)
	if err != nil {
		return nil, err
	}

	c := &Tasmota{
		conn:  conn,
		usage: usage,
	}

	return c, nil
}

var _ api.Meter = (*Tasmota)(nil)

// CurrentPower implements the api.Meter interface
func (c *Tasmota) CurrentPower() (float64, error) {
	if c.usage == "grid" {
		return c.conn.SmlPower()
	}
	return c.conn.CurrentPower()
}

var _ api.MeterEnergy = (*Tasmota)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Tasmota) TotalEnergy() (float64, error) {
	if c.usage == "grid" {
		return c.conn.SmlTotalEnergy()
	}
	return c.conn.TotalEnergy()
}

var _ api.PhaseCurrents = (*Tasmota)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *Tasmota) Currents() (float64, float64, float64, error) {
	if c.usage == "grid" {
		return 0, 0, 0, nil
	}
	return c.conn.Currents()
}
