package meter

import (
	"math"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/meter/shelly"
	"github.com/evcc-io/evcc/util"
)

// Shelly meter considering usage
type Shelly struct {
	implement.Caps
	conn  *shelly.Connection
	usage string
}

// Shelly meter implementation
func init() {
	registry.Add("shelly", NewShellyFromConfig)
}

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

	// Three-phase Shelly energy meters count each phase separately (non-balanced),
	// making their totals unsuitable for bidirectional grid metering.
	if !(c.usage == "grid" && c.conn.IsThreePhase()) {
		total, ret := c.conn.TotalEnergy, c.conn.ReturnEnergy

		hasReturn := c.conn.HasReturnEnergy()

		// production is measured in return direction, unless the device has no return
		// register at all or already reverses the direction itself
		if c.usage == "pv" && hasReturn && !c.conn.IsReversed() {
			total, ret = ret, total
		}
		implement.Has(c, implement.MeterEnergy(total))

		// without a return register the second reading is a constant zero
		if hasReturn {
			implement.Has(c, implement.MeterReturnEnergy(ret))
		}
	}

	if phases, ok := c.conn.Generation.(shelly.Phases); ok {
		implement.Has(c, implement.PhaseVoltages(phases.Voltages))
		implement.Has(c, implement.PhaseCurrents(phases.Currents))
		implement.Has(c, implement.PhasePowers(phases.Powers))
	}

	return c, nil
}

// NewShelly creates Shelly meter
func NewShelly(uri, user, password, usage string, channel int, cache time.Duration) (*Shelly, error) {
	conn, err := shelly.NewConnection(uri, user, password, channel, cache)
	if err != nil {
		return nil, err
	}
	c := &Shelly{
		Caps:  implement.New(),
		conn:  conn,
		usage: usage,
	}
	return c, nil
}

var _ api.Meter = (*Shelly)(nil)

// CurrentPower implements the api.Meter interface
func (c *Shelly) CurrentPower() (float64, error) {
	power, err := c.conn.CurrentPower()
	if err != nil {
		return 0, err
	}
	return c.currentPowerForUsage(power, c.conn.SignedPower(), c.conn.IsReversed()), nil
}

// PV usage inverts directional power unless the device already reverses it, otherwise the magnitude is used.
func (c *Shelly) currentPowerForUsage(power float64, signed, reversed bool) float64 {
	if c.usage != "pv" {
		return power
	}
	switch {
	case !signed:
		return math.Abs(power)
	case reversed:
		return power
	default:
		return -power
	}
}
