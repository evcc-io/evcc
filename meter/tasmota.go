package meter

import (
	"math"
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

//go:generate go tool decorate -f decorateTasmota -b *Tasmota -r api.Meter -t api.PhaseVoltages,api.PhaseCurrents,api.PhasePowers

// NewTasmotaFromConfig creates a Tasmota meter from generic config
func NewTasmotaFromConfig(other map[string]any) (api.Meter, error) {
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

	// check for SML readings
	var hasPhases bool
	if len(channels) == 1 {
		if l1, l2, l3, err := c.conn.Voltages(); err == nil && l1*l2*l3 > 0 {
			hasPhases = true
		}
	}

	if hasPhases || len(channels) == 3 {
		return decorateTasmota(c, c.voltages, c.currents, c.powers), nil
	}

	return c, nil
}

var _ api.Meter = (*Tasmota)(nil)

// CurrentPower implements the api.Meter interface
func (c *Tasmota) CurrentPower() (float64, error) {
	power, err := c.conn.CurrentPower()
	if err != nil {
		return 0, err
	}
	// positive power for pv usage
	if c.usage == "pv" {
		return math.Abs(power), nil
	}
	return power, nil
}

var _ api.MeterEnergy = (*Tasmota)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Tasmota) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}

// powers implements the api.PhasePowers interface
func (c *Tasmota) powers() (float64, float64, float64, error) {
	return c.conn.Powers()
}

// voltages implements the api.PhaseVoltages interface
func (c *Tasmota) voltages() (float64, float64, float64, error) {
	return c.conn.Voltages()
}

// currents implements the api.PhaseCurrents interface
func (c *Tasmota) currents() (float64, float64, float64, error) {
	return c.conn.Currents()
}
