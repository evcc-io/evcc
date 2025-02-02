package meter

import (
	"strings"
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

//go:generate go run ../cmd/tools/decorate.go -f decorateTasmota -b *Tasmota -r api.Meter -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)"

// NewTasmotaFromConfig creates a Tasmota meter from generic config
func NewTasmotaFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI      string
		User     string
		Password string
		Channel  []int
		Usage    string
		Cache    time.Duration
	}{
		Channel: []int{1},
		Cache:   time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewTasmota(cc.URI, cc.User, cc.Password, strings.ToLower(cc.Usage), cc.Channel, cc.Cache)
}

// NewTasmota creates Tasmota meter
func NewTasmota(uri, user, password, usage string, channels []int, cache time.Duration) (api.Meter, error) {
	conn, err := tasmota.NewConnection(uri, user, password, channels, cache)
	if err != nil {
		return nil, err
	}

	c := &Tasmota{
		conn:  conn,
		usage: usage,
	}

	var currents, voltages func() (float64, float64, float64, error)
	if usage != "grid" && len(channels) == 3 {
		currents = c.currents
		voltages = c.voltages
	}

	return decorateTasmota(c, currents, voltages), nil
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

// currents implements the api.PhaseCurrents interface
func (c *Tasmota) currents() (float64, float64, float64, error) {
	return c.conn.Currents()
}

// voltages implements the api.PhaseVoltages interface
func (c *Tasmota) voltages() (float64, float64, float64, error) {
	return c.conn.Voltages()
}
