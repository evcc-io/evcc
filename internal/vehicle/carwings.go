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
	hvacG          func() (interface{}, error)
}

func init() {
	registry.Add("carwings", NewCarWingsFromConfig)
}

// NewCarWingsFromConfig creates a new vehicle
func NewCarWingsFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed                  `mapstructure:",squash"`
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
		embed:    &cc.embed,
		user:     cc.User,
		password: cc.Password,
		region:   cc.Region,
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()
	v.hvacG = provider.NewCached(v.hvacAPI, cc.Cache).InterfaceGetter()

	return v, nil
}

// Init new carwings session
func (v *CarWings) initSession() error {
	v.session = &carwings.Session{
		Region: v.region,
	}

	return v.session.Connect(v.user, v.password)
}

// chargeState implements the api.Vehicle interface
func (v *CarWings) chargeState() (float64, error) {
	if v.session == nil {
		if err := v.initSession(); err != nil {
			return 0, api.ErrNotAvailable
		}
	}

	bs, err := v.session.BatteryStatus()
	return float64(bs.StateOfCharge), err
}

// hvacAPI provides hvac-status api response
func (v *CarWings) hvacAPI() (interface{}, error) {
	if v.session == nil {
		if err := v.initSession(); err != nil {
			return 0, api.ErrNotAvailable
		}
	}

	return v.session.ClimateControlStatus()
}

// SoC implements the api.Vehicle interface
func (v *CarWings) SoC() (float64, error) {
	return v.chargeStateG()
}
