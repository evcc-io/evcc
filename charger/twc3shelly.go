package charger

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/shelly"
	"github.com/evcc-io/evcc/util"
)

// Twc3Shelly extends Twc3 with 1p3p phase switching via a Shelly switch that
// controls the L2/L3 contactor. L1 must be permanently connected to the TWC3.
//
//	evcc.yaml:
//	  charger:
//	    type: twc3shelly
//	    host: 192.168.1.20
//	    shelly: 192.168.1.100
type Twc3Shelly struct {
	*Twc3
	shellyConn *shelly.Connection
}

// compile-time interface assertions
var (
	_ api.Charger       = (*Twc3Shelly)(nil)
	_ api.PhaseSwitcher = (*Twc3Shelly)(nil)
)

func init() {
	registry.Add("twc3shelly", NewTwc3ShellyFromConfig)
}

// NewTwc3ShellyFromConfig creates a new TWC3 charger with Shelly phase switching
func NewTwc3ShellyFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		URI          string
		Cache        time.Duration
		Shelly       string
		ShellyUser   string
		ShellyPass   string
		ShellySwitch int
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.Shelly == "" {
		return nil, fmt.Errorf("missing shelly uri")
	}

	inner, err := NewTwc3FromConfig(map[string]any{"uri": cc.URI, "cache": cc.Cache})
	if err != nil {
		return nil, err
	}

	conn, err := shelly.NewConnection(cc.Shelly, cc.ShellyUser, cc.ShellyPass, cc.ShellySwitch, cc.Cache)
	if err != nil {
		return nil, err
	}

	return &Twc3Shelly{
		Twc3:       inner.(*Twc3),
		shellyConn: conn,
	}, nil
}

// Phases1p3p implements the api.PhaseSwitcher interface
func (c *Twc3Shelly) Phases1p3p(phases int) error {
	on := phases == 3
	if !on && phases != 1 {
		return fmt.Errorf("invalid phase count %d", phases)
	}

	// safety: never switch the L2/L3 contactor while a significant vehicle
	// current is flowing
	res, err := c.vitalsG()
	if err != nil {
		return fmt.Errorf("reading twc3 vitals: %w", err)
	}
	if res.VehicleCurrentA > 1.0 {
		return fmt.Errorf("vehicle current %.1fA too high for phase switch", res.VehicleCurrentA)
	}

	if err := c.shellyConn.Enable(on); err != nil {
		return fmt.Errorf("shelly switch %v: %w", on, err)
	}

	// verify that the shelly confirmed the new output state
	var got bool
	for i := 0; i < 3; i++ {
		if got, err = c.shellyConn.Enabled(); err == nil && got == on {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("verifying shelly state: %w", err)
	}
	return fmt.Errorf("shelly switch state %v != expected %v", got, on)
}
