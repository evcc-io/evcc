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
	shelly.Connection
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

	c, err := NewShelly(cc.URI, cc.User, cc.Password, strings.ToLower(cc.Usage), cc.Channel, cc.Cache)
	if err != nil {
		return nil, err
	}

	var vol, cur, pow func() (float64, float64, float64, error)
	if phases, ok := c.Connection.Generation.(shelly.Phases); ok {
		vol = phases.Voltages
		cur = phases.Currents
		pow = phases.Powers
	}

	return decorateShelly(c, vol, cur, pow), nil
}

// NewShelly creates Shelly meter
func NewShelly(uri, user, password, usage string, channel int, cache time.Duration) (*Shelly, error) {
	conn, err := shelly.NewConnection(uri, user, password, channel, cache)
	if err != nil {
		return nil, err
	}
	c := &Shelly{
		Connection: *conn,
		usage:      usage,
	}
	return c, nil
}

var _ api.Meter = (*Shelly)(nil)

// CurrentPower implements the api.Meter interface
func (c *Shelly) CurrentPower() (float64, error) {
	power, err := c.Connection.CurrentPower()
	if err != nil {
		return 0, err
	}
	if c.usage == "pv" {
		power = math.Abs(power)
	}
	return power, nil
}
