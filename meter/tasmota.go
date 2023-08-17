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
		Usage    string
		Cache    time.Duration
	}{
		Channel: 1,
		Cache:   time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewTasmota(cc.URI, cc.User, cc.Password, cc.Usage, cc.Channel, cc.Cache)
}

// NewTasmota creates Tasmota meter
func NewTasmota(uri, user, password, usage string, channel int, cache time.Duration) (*Tasmota, error) {
	conn, err := tasmota.NewConnection(uri, user, password, channel, cache)
	if err != nil {
		return nil, err
	}

	c := &Tasmota{
		conn:  conn,
		usage: usage,
	}

	return c, err
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
