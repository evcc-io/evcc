package meter

import (
	"math"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/meter/tasmota"
	"github.com/evcc-io/evcc/util"
)

// Tasmota meter implementation
type Tasmota struct {
	implement.Caps
	conn  *tasmota.Connection
	usage string
}

// Tasmota meter implementation
func init() {
	registry.Add("tasmota", NewTasmotaFromConfig)
}

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
		Caps:  implement.New(),
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
		implement.Has(c, implement.PhaseVoltages(c.voltages))
		implement.Has(c, implement.PhaseCurrents(c.currents))
		implement.Has(c, implement.PhasePowers(c.powers))
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

var _ api.MeterImport = (*Tasmota)(nil)

// ImportEnergy implements the api.MeterImport interface
func (c *Tasmota) ImportEnergy() (float64, error) {
	return c.conn.ImportEnergy()
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
