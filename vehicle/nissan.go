package vehicle

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"

	"github.com/joeshaw/carwings"
)

// Nissan is an api.Vehicle implementation for Nissan cars
type Nissan struct {
	*embed
	session      *carwings.Session
	chargeStateG func() (float64, error)
}

// NewNissanFromConfig creates a new vehicle
func NewNissanFromConfig(log *util.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title                  string
		Capacity               int64
		User, Password, Region string
		Cache                  time.Duration
	}{}
	util.DecodeOther(log, other, &cc)

	if cc.Region == "" {
		cc.Region = carwings.RegionEurope
	}

	session := &carwings.Session{
		Region: cc.Region,
	}

	if err := session.Connect(cc.User, cc.Password); err != nil {
		log.FATAL.Fatalf("cannot create nissan: %v", err)
	}

	v := &Nissan{
		embed:   &embed{cc.Title, cc.Capacity},
		session: session,
	}

	v.chargeStateG = provider.NewCached(log, v.chargeState, cc.Cache).FloatGetter()

	return v
}

// chargeState implements the Vehicle.ChargeState interface
func (v *Nissan) chargeState() (float64, error) {
	bs, err := v.session.BatteryStatus()
	return float64(bs.StateOfCharge), err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Nissan) ChargeState() (float64, error) {
	return v.chargeStateG()
}
