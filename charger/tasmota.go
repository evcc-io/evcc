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

//go:generate go run ../cmd/tools/decorate.go -f decorateTasmota -b *Tasmota -r api.Charger -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)"

// NewTasmotaFromConfig creates a Tasmota charger from generic config
func NewTasmotaFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		embed        `mapstructure:",squash"`
		URI          string
		User         string
		Password     string
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

	return NewTasmota(cc.embed, cc.URI, cc.User, cc.Password, cc.Channel, cc.StandbyPower, cc.Cache)
}

// NewTasmota creates Tasmota charger
func NewTasmota(embed embed, uri, user, password string, channels []int, standbypower float64, cache time.Duration) (api.Charger, error) {
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

	var currents, voltages func() (float64, float64, float64, error)
	if len(channels) == 3 {
		currents = c.currents
		voltages = c.voltages
	}

	return decorateTasmota(c, currents, voltages), nil
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
