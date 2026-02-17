package charger

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/shelly"
	"github.com/evcc-io/evcc/util"
)

// Shelly charger implementation
type Shelly struct {
	conn *shelly.Connection
	*switchSocket
}

func init() {
	registry.Add("shelly", NewShellyFromConfig)
}

//go:generate go tool decorate -f decorateShelly -b *Shelly -r api.Charger -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhasePowers,Powers,func() (float64, float64, float64, error)"

// NewShellyFromConfig creates a Shelly charger from generic config
func NewShellyFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		embed        `mapstructure:",squash"`
		URI          string
		User         string
		Password     string
		Channel      int
		StandbyPower float64
		Cache        time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	c, err := NewShelly(cc.embed, cc.URI, cc.User, cc.Password, cc.Channel, cc.StandbyPower, cc.Cache)
	if err != nil {
		return nil, err
	}

	var vol, cur, pow func() (float64, float64, float64, error)
	if phases, ok := c.conn.Generation.(shelly.Phases); ok {
		vol = phases.Voltages
		cur = phases.Currents
		pow = phases.Powers
	}

	return decorateShelly(c, vol, cur, pow), nil
}

// NewShelly creates Shelly charger
func NewShelly(embed embed, uri, user, password string, channel int, standbypower float64, cache time.Duration) (*Shelly, error) {
	conn, err := shelly.NewConnection(uri, user, password, channel, cache)
	if err != nil {
		return nil, err
	}

	c := &Shelly{
		conn: conn,
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.conn.CurrentPower, standbypower)

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *Shelly) Enabled() (bool, error) {
	return c.conn.Enabled()
}

// Enable implements the api.Charger interface
func (c *Shelly) Enable(enable bool) error {
	if err := c.conn.Enable(enable); err != nil {
		return err
	}

	enabled, err := c.Enabled()
	switch {
	case err != nil:
		return err
	case enable != enabled:
		onoff := map[bool]string{true: "on", false: "off"}
		return fmt.Errorf("switch %s failed", onoff[enable])
	default:
		return nil
	}
}

var _ api.MeterEnergy = (*Shelly)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Shelly) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}
