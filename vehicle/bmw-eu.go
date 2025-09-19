package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	bmw "github.com/evcc-io/evcc/vehicle/bmw/eu"
	"golang.org/x/oauth2"
)

// BMWEU is an api.Vehicle implementation for BMW and Mini cars
type BMWEU struct {
	*embed
	*bmw.Provider // provides the api implementations
}

func init() {
	registry.Add("bmw-eu", NewBMWEUFromConfig)
	registry.Add("mini-eu", NewMiniEUFromConfig)
}

// NewBMWEUFromConfig creates a new vehicle
func NewBMWEUFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	return NewBMWMiniEU("bmw-eu", bmw.Config, other)
}

// NewMiniEUFromConfig creates a new vehicle
func NewMiniEUFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	return NewBMWMiniEU("mini-eu", bmw.Config, other)
}

// NewBMWMiniEU creates a new vehicle
func NewBMWMiniEU(brand string, config oauth2.Config, other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed         `mapstructure:",squash"`
		ClientID, VIN string
		Cache         time.Duration
	}{
		Cache: 30 * time.Minute, // 50 requests per day
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientID == "" {
		return nil, api.ErrMissingCredentials
	}
	config.ClientID = cc.ClientID

	v := &BMWEU{
		embed: &cc.embed,
	}

	log := util.NewLogger(brand).Redact(cc.ClientID)
	identity := bmw.NewIdentity(log, &config)

	ts, err := identity.Login()
	if err != nil {
		return nil, err
	}

	api := bmw.NewAPI(log, brand, cc.Region, ts)

	vehicle, err := ensureVehicleEx(
		cc.VIN, api.Vehicles,
		func(v bmw.Vehicle) (string, error) {
			return v.VIN, nil
		},
	)

	if err == nil {
		v.Provider = bmw.NewProvider(api, vehicle.VIN, cc.Cache)
	}

	return v, err
}
