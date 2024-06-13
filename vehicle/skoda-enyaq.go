package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/skoda/myskoda"
	"github.com/evcc-io/evcc/vehicle/skoda/myskoda/service"
)

// https://gitlab.com/prior99/skoda

// Enyaq is an api.Vehicle implementation for Skoda Enyaq cars
type Enyaq struct {
	*embed
	*myskoda.Provider // provides the api implementations
}

func init() {
	registry.Add("enyaq", NewEnyaqFromConfig)
}

// NewEnyaqFromConfig creates a new vehicle
func NewEnyaqFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
		Timeout             time.Duration
	}{
		Cache:   interval,
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	v := &Enyaq{
		embed: &cc.embed,
	}

	var err error
	log := util.NewLogger("enyaq").Redact(cc.User, cc.Password, cc.VIN)

	// use Skoda api to resolve list of vehicles
	ts, err := service.TokenRefreshServiceTokenSource(log, myskoda.TRSParams, myskoda.AuthParams, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	api := myskoda.NewAPI(log, ts)
	api.Client.Timeout = cc.Timeout

	vehicle, err := ensureVehicleEx(
		cc.VIN, api.Vehicles,
		func(v myskoda.Vehicle) (string, error) {
			return v.VIN, nil
		},
	)

	if err == nil {
		vehicle, err = api.VehicleDetails(vehicle.VIN)
	}

	if err == nil {
		v.fromVehicle(vehicle.Name, float64(vehicle.Specification.Battery.CapacityInKWh))
	}

	// reuse tokenService to build provider
	if err == nil {
		api := myskoda.NewAPI(log, ts)
		api.Client.Timeout = cc.Timeout

		v.Provider = myskoda.NewProvider(api, vehicle.VIN, cc.Cache)
	}

	return v, err
}
