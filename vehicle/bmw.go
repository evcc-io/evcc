package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/bmw"
)

// BMW is an api.Vehicle implementation for BMW and Mini cars
type BMW struct {
	*embed
	*bmw.Provider // provides the api implementations
}

func init() {
	registry.Add("bmw", NewBMWFromConfig)
	registry.Add("mini", NewMiniFromConfig)
}

// NewBMWFromConfig creates a new vehicle
func NewBMWFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	return NewBMWMiniFromConfig("bmw", other)
}

// NewMiniFromConfig creates a new vehicle
func NewMiniFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	return NewBMWMiniFromConfig("mini", other)
}

// NewBMWMiniFromConfig creates a new vehicle
func NewBMWMiniFromConfig(brand string, other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Region              string
		Cache               time.Duration
	}{
		Region: "EU",
		Cache:  interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	v := &BMW{
		embed: &cc.embed,
	}

	log := util.NewLogger(brand).Redact(cc.User, cc.Password, cc.VIN)
	identity := bmw.NewIdentity(log, cc.Region)

	ts, err := identity.Login(cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	api := bmw.NewAPI(log, brand, cc.Region, ts)

	vehicle, err := ensureVehicleEx(
		cc.VIN, api.Vehicles,
		func(v bmw.Vehicle) string {
			return v.VIN
		},
	)

	if err == nil {
		v.Provider = bmw.NewProvider(api, vehicle.VIN, cc.Cache)
	}

	return v, err
}
