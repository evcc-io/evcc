package meter

import (
	"math"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/shelly"
	"github.com/evcc-io/evcc/util"
)

// Shelly meter considering usage
type Shelly struct {
	conn  *shelly.Connection
	usage string
}

// Shelly meter implementation
func init() {
	registry.Add("shelly", NewShellyFromConfig)
}

//go:generate go tool decorate -f decorateShelly -b *Shelly -r api.Meter -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhasePowers,Powers,func() (float64, float64, float64, error)"

// NewShellyFromConfig creates a Shelly charger from generic config
func NewShellyFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		URI      string
		User     string
		Password string
		Channel  int
		Usage    string
		Cache    time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewShelly(cc.URI, cc.User, cc.Password, strings.ToLower(cc.Usage), cc.Channel, cc.Cache)
}

// NewShelly creates Shelly meter
func NewShelly(uri, user, password, usage string, channel int, cache time.Duration) (api.Meter, error) {
	conn, err := shelly.NewConnection(uri, user, password, channel, cache)
	if err != nil {
		return nil, err
	}

	c := &Shelly{
		conn:  conn,
		usage: usage,
	}

	var currents, voltages, powers func() (float64, float64, float64, error)
	currents = c.currents
	voltages = c.voltages
	powers = c.powers

	return decorateShelly(c, voltages, currents, powers), nil
}

var _ api.Meter = (*Shelly)(nil)

// CurrentPower implements the api.Meter interface
func (c *Shelly) CurrentPower() (float64, error) {
	power, err := c.conn.CurrentPower()
	if err != nil {
		return 0, err
	}
	if c.usage == "pv" {
		power = math.Abs(power)
	}
	return power, nil
}

var _ api.MeterEnergy = (*Shelly)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Shelly) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}

// currents implements the api.PhaseCurrents interface
func (c *Shelly) currents() (float64, float64, float64, error) {
	return c.conn.Currents()
}

// voltages implements the api.PhaseVoltages interface
func (c *Shelly) voltages() (float64, float64, float64, error) {
	return c.conn.Voltages()
}

// powers implements the api.PhaseVoltages interface
func (c *Shelly) powers() (float64, float64, float64, error) {
	return c.conn.Powers()
}
