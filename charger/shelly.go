package charger

import (
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

// NewShellyFromConfig creates a Shelly charger from generic config
func NewShellyFromConfig(other map[string]interface{}) (api.Charger, error) {
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

	return NewShelly(cc.embed, cc.URI, cc.User, cc.Password, cc.Channel, cc.StandbyPower, cc.Cache)
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

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.CurrentPower, standbypower)

	return c, nil
}

var _ api.Meter = (*Shelly)(nil)

// CurrentPower implements the api.Meter interface
func (c *Shelly) CurrentPower() (float64, error) {
	switch c.conn.Gen {
	case 0, 1:
		return c.conn.Gen1CurrentPower()
	default:
		return c.conn.Gen2CurrentPower()
	}
}

// Enabled implements the api.Charger interface
func (c *Shelly) Enabled() (bool, error) {
	switch c.conn.Gen {
	case 0, 1:
		return c.conn.Gen1Enabled()
	default:
		return c.conn.Gen2Enabled()
	}
}

// Enable implements the api.Charger interface
func (c *Shelly) Enable(enable bool) error {
	switch c.conn.Gen {
	case 0, 1:
		return c.conn.Gen1Enable(enable)
	default:
		return c.conn.Gen2Enable(enable)
	}
}

var _ api.MeterEnergy = (*Shelly)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Shelly) TotalEnergy() (float64, error) {
	var energy float64
	var err error
	switch c.conn.Gen {
	case 0, 1:
		energy, err = c.conn.Gen1TotalEnergy()
		if err != nil {
			return 0, err
		}

	default:
		energy, _, err = c.conn.Gen2TotalEnergy()
		if err != nil {
			return 0, err
		}
	}
	return energy / 1000, nil
}
