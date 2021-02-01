package vehicle

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"

	"github.com/joeshaw/carwings"
)

// CarWings is an api.Vehicle implementation for CarWings cars
type CarWings struct {
	*embed
	user, password string
	region         string
	session        *carwings.Session
	chargeStateG   func() (float64, error)
}

func init() {
	registry.Add("carwings", NewCarWingsFromConfig)
}

// NewCarWingsFromConfig creates a new vehicle
func NewCarWingsFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title                  string
		Capacity               int64
		User, Password, Region string
		Cache                  time.Duration
	}{
		Region: carwings.RegionEurope,
		Cache:  interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &CarWings{
		embed:    &embed{cc.Title, cc.Capacity},
		user:     cc.User,
		password: cc.Password,
		region:   cc.Region,
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	return v, nil
}

// chargeState implements the api.Vehicle interface
func (v *CarWings) chargeState() (float64, error) {
	if v.session == nil {
		session := &carwings.Session{
			Region: v.region,
		}

		if err := session.Connect(v.user, v.password); err != nil {
			return 0, err
		}

		v.session = session
	}

	bs, err := v.session.BatteryStatus()
	return float64(bs.StateOfCharge), err
}

// SoC implements the api.Vehicle interface
func (v *CarWings) SoC() (float64, error) {
	return v.chargeStateG()
}
