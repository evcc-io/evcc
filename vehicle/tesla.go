package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/evcc-io/evcc/vehicle/tesla"
)

// Tesla is an api.Vehicle implementation for Tesla cars using the official Tesla vehicle-command api.
type Tesla struct {
	*embed
	*tesla.Provider
	*tesla.Controller
}

func init() {
	registry.Add("tesla", NewTeslaFromConfig)
}

// NewTeslaFromConfig creates a new vehicle
func NewTeslaFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed             `mapstructure:",squash"`
		tesla.FleetConfig `mapstructure:",squash"`
		VIN               string
		CommandProxy      string
		ProxyToken        string
		Cache             time.Duration
		Timeout           time.Duration
	}{
		CommandProxy: tesla.ProxyBaseUrl,
		Cache:        interval,
		Timeout:      request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if err := cc.FleetConfig.Validate(); err != nil {
		return nil, err
	}

	log := util.NewLogger("tesla").Redact(
		cc.Tokens.Access, cc.Tokens.Refresh, cc.ProxyToken,
		cc.Credentials.ID, cc.Credentials.Secret,
	)

	fleet, err := cc.FleetConfig.Client(log)
	if err != nil {
		return nil, err
	}

	vehicle, err := ensureVehicleEx(
		cc.VIN, fleet.Vehicles,
		func(v tesla.Vehicle) (string, error) {
			return v.Vin, nil
		},
	)
	if err != nil {
		return nil, err
	}

	// proxy client
	pc := request.NewClient(log)
	pc.Transport = &transport.Decorator{
		Decorator: transport.DecorateHeaders(map[string]string{
			"X-Authorization": "Bearer " + cc.ProxyToken,
		}),
		Base: fleet.Transport,
	}

	v := &Tesla{
		embed:      &cc.embed,
		Provider:   tesla.NewProvider(fleet, vehicle.Vin, cc.Cache),
		Controller: tesla.NewController(tesla.NewAPI(pc, cc.CommandProxy), vehicle.Vin),
	}

	v.fromVehicle(vehicle.DisplayName, 0)

	return v, nil
}
