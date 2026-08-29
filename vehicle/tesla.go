package vehicle

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/evcc-io/evcc/vehicle/tesla"
	teslaclient "github.com/evcc-io/tesla-proxy-client"
)

type teslaController interface {
	api.CurrentController
	api.ChargeController
}

// Tesla is an api.Vehicle implementation for Tesla cars using the official Tesla vehicle-command api.
type Tesla struct {
	*embed
	*tesla.Provider
	teslaController
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
	tc := fleet.Client

	vehicle, err := ensureVehicleEx(
		cc.VIN, tc.Vehicles,
		func(v *tesla.Vehicle) (string, error) {
			return v.Vin, nil
		},
	)
	if err != nil {
		return nil, err
	}

	var controller teslaController

	if cc.Interactive() {
		key, err := tesla.SigningKey()
		if err != nil {
			return nil, err
		}

		controller = tesla.NewSignedController(fleet.TokenSource, key, vehicle, fleet.Host)
	} else {
		// proxy client
		pc := request.NewClient(log)
		pc.Transport = &transport.Decorator{
			Decorator: transport.DecorateHeaders(map[string]string{
				"X-Authorization": "Bearer " + cc.ProxyToken,
			}),
			Base: fleet.HTTPClient.Transport,
		}

		tcc, err := teslaclient.NewClient(context.Background(), teslaclient.WithClient(pc))
		if err != nil {
			return nil, err
		}
		tcc.SetBaseUrl(cc.CommandProxy)

		controller = tesla.NewController(vehicle.WithClient(tcc))
	}

	v := &Tesla{
		embed:           &cc.embed,
		Provider:        tesla.NewProvider(vehicle, cc.Cache),
		teslaController: controller,
	}

	v.fromVehicle(vehicle.DisplayName, 0)

	return v, nil
}
