package meter

import (
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/shelly"
	"github.com/evcc-io/evcc/util"
)

type Shelly struct {
	conn  *shelly.Connection
	usage string
}

// Shelly meter implementation
func init() {
	registry.Add("shelly", NewShellyFromConfig)
}

// NewShellyFromConfig creates a Shelly charger from generic config
func NewShellyFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI      string
		User     string
		Password string
		Channel  int
		Usage    string // grid, pv, battery
		Cache    time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	conn, err := shelly.NewConnection(cc.URI, cc.User, cc.Password, cc.Channel, cc.Cache)
	if err != nil {
		return nil, err
	}

	return NewShelly(conn, strings.ToLower(cc.Usage)), nil
}

func NewShelly(conn *shelly.Connection, usage string) *Shelly {
	res := &Shelly{
		conn:  conn,
		usage: usage,
	}

	return res
}

var _ api.Meter = (*Shelly)(nil)

// CurrentPower implements the api.Meter interface
func (c *Shelly) CurrentPower() (float64, error) {
	var power float64
	var err error

	switch c.conn.Gen {
	case 0, 1:
		power, err = c.conn.Gen1CurrentPower()
		if err != nil {
			return 0, err
		}
	default:
		power, err = c.conn.Gen2CurrentPower()
		if err != nil {
			return 0, err
		}
	}

	// Asure positive power values for pv usage
	if c.usage == "pv" && power < 0 {
		return -power, nil
	}

	return power, nil
}

var _ api.MeterEnergy = (*Shelly)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Shelly) TotalEnergy() (float64, error) {
	var energy float64
	var energyConsumed float64
	var energyFeedIn float64
	var err error

	switch c.conn.Gen {
	case 0, 1:
		energyConsumed, err = c.conn.Gen1TotalEnergy()
		if err != nil {
			return 0, err
		}
	default:
		energyConsumed, energyFeedIn, err = c.conn.Gen2TotalEnergy()
		if err != nil {
			return 0, err
		}
	}

	if c.usage == "pv" || c.usage == "battery" {
		energy = energyFeedIn
	} else {
		energy = energyConsumed
	}

	return energy, nil
}

var _ api.PhaseCurrents = (*Shelly)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *Shelly) Currents() (float64, float64, float64, error) {
	switch c.conn.Gen {
	case 0, 1:
		return c.conn.Gen1Currents()
	default:
		return c.conn.Gen2Currents()
	}
}

var _ api.PhaseVoltages = (*Shelly)(nil)

// Voltages implements the api.PhaseVoltages interface
func (c *Shelly) Voltages() (float64, float64, float64, error) {
	switch c.conn.Gen {
	case 0, 1:
		return c.conn.Gen1Voltages()
	default:
		return c.conn.Gen2Voltages()
	}
}

var _ api.PhasePowers = (*Shelly)(nil)

// Powers implements the api.PhasePowers interface
func (c *Shelly) Powers() (float64, float64, float64, error) {
	switch c.conn.Gen {
	case 0, 1:
		return c.conn.Gen1Powers()
	default:
		return c.conn.Gen2Powers()
	}
}
