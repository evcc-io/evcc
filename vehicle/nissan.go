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

func init() {
	registry.Add("nissan", NewNissanFromConfig)
}

// NewNissanFromConfig creates a new vehicle
func NewNissanFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title          string
		Capacity       int64
		User, Password string `validate:"required"`
		Region         string
		Cache          time.Duration
	}{
		Region: carwings.RegionEurope,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	session := &carwings.Session{
		Region: cc.Region,
	}

	if err := session.Connect(cc.User, cc.Password); err != nil {
		return nil, err
	}

	v := &Nissan{
		embed:   &embed{cc.Title, cc.Capacity},
		session: session,
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	return v, nil
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
