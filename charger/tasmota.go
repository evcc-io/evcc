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
	conn *tasmota.Connection
	*switchSocket
}

func init() {
	registry.Add("tasmota", NewTasmotaFromConfig)
}

//go:generate go tool decorate -f decorateTasmota -b *Tasmota -r api.Charger -t api.PhaseVoltages,api.PhaseCurrents

// NewTasmotaFromConfig creates a Tasmota charger from generic config
func NewTasmotaFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		embed        `mapstructure:",squash"`
		URI          string
		User         string
		Password     string
		Usage        string
		StandbyPower float64
		Channel      []int
		Cache        time.Duration
	}{
		Channel: []int{1},
		Cache:   time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewTasmota(cc.embed, cc.URI, cc.User, cc.Password, cc.Usage, cc.Channel, cc.StandbyPower, cc.Cache)
}

// NewTasmota creates Tasmota charger
func NewTasmota(embed embed, uri, user, password, usage string, channels []int, standbypower float64, cache time.Duration) (api.Charger, error) {
	conn, err := tasmota.NewConnection(uri, user, password, channels, cache)
	if err != nil {
		return nil, err
	}

	if err := conn.RelayExists(); err != nil {
		return nil, err
	}

	c := &Tasmota{
		conn: conn,
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.conn.CurrentPower, standbypower)

	// check if phase specific readings are supported by the device, if not return the base meter implementation without decorators
	var hasPhases bool
	if len(channels) == 1 {
		if l1, l2, l3, err := c.conn.Voltages(); err == nil && l1*l2*l3 > 0 {
			hasPhases = true
		}
	}

	if hasPhases || len(channels) == 3 {
		return decorateTasmota(c, c.voltages, c.currents), nil
	}

	return c, nil
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

// Currents implements the api.PhaseCurrents interface
func (c *Tasmota) currents() (float64, float64, float64, error) {
	return c.conn.Currents()
}

// Voltages implements the api.PhaseVoltages interface
func (c *Tasmota) voltages() (float64, float64, float64, error) {
	return c.conn.Voltages()
}
