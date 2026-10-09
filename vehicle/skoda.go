package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/skoda"
	"github.com/evcc-io/evcc/vehicle/skoda/service"
)

// https://gitlab.com/prior99/skoda

// Skoda is an api.Vehicle implementation for Skoda cars
type Skoda struct {
	*embed
	*skoda.Provider // provides the api implementations
}

func init() {
	registry.Add("skoda", NewSkodaFromConfig)
}

// NewSkodaFromConfig creates a new vehicle
func NewSkodaFromConfig(other map[string]any) (api.Vehicle, error) {
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

	v := &Skoda{
		embed: &cc.embed,
	}

	log := util.NewLogger("skoda").Redact(cc.User, cc.Password, cc.VIN)

	// use Skoda api to resolve list of vehicles
	ts, err := service.TokenRefreshServiceTokenSource(log, skoda.TRSParams, skoda.AuthParams, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	client := skoda.NewAPI(log, ts)
	client.Client.Timeout = cc.Timeout

	vehicle, err := ensureVehicleEx(
		cc.VIN, client.Vehicles,
		func(v skoda.Vehicle) (string, error) {
			return v.VIN, nil
		},
	)
	if err != nil {
		return nil, err
	}

	if vehicle, err = client.VehicleDetails(vehicle.VIN); err != nil {
		return nil, err
	}

	v.fromVehicle(vehicle.Name, float64(vehicle.Specification.Battery.CapacityInKWh))
	v.Provider = skoda.NewProvider(client, vehicle.VIN, cc.Cache)

	return v, nil
}
