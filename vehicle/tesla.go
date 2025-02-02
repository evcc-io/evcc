package vehicle

import (
	"context"
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/evcc-io/evcc/vehicle/tesla"
	teslaclient "github.com/evcc-io/tesla-proxy-client"
	"golang.org/x/oauth2"
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
func NewTeslaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed        `mapstructure:",squash"`
		Credentials  ClientCredentials
		Tokens       Tokens
		VIN          string
		CommandProxy string
		ProxyToken   string
		Cache        time.Duration
		Timeout      time.Duration
	}{
		CommandProxy: tesla.ProxyBaseUrl,
		Cache:        interval,
		Timeout:      request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Credentials.ID == "" {
		return nil, errors.New("missing client id, see https://github.com/evcc-io/evcc/discussions/17501")
	}

	token, err := cc.Tokens.Token()
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("tesla").Redact(
		cc.Tokens.Access, cc.Tokens.Refresh, cc.ProxyToken,
		cc.Credentials.ID, cc.Credentials.Secret,
	)

	identity, err := tesla.NewIdentity(log, tesla.OAuth2Config(cc.Credentials.ID, cc.Credentials.Secret), token)
	if err != nil {
		return nil, err
	}

	hc := request.NewClient(log)
	baseTransport := hc.Transport

	hc.Transport = &oauth2.Transport{
		Source: identity,
		Base:   baseTransport,
	}

	tc, err := teslaclient.NewClient(context.Background(), teslaclient.WithClient(hc))
	if err != nil {
		return nil, err
	}

	// validate base url
	region, err := tc.UserRegion()
	if err != nil {
		return nil, err
	}
	tc.SetBaseUrl(region.FleetApiBaseUrl)

	vehicle, err := ensureVehicleEx(
		cc.VIN, tc.Vehicles,
		func(v *tesla.Vehicle) (string, error) {
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
			"Authorization": "Bearer " + cc.ProxyToken,
		}),
		Base: baseTransport,
	}

	tcc, err := teslaclient.NewClient(context.Background(), teslaclient.WithClient(pc))
	if err != nil {
		return nil, err
	}
	tcc.SetBaseUrl(cc.CommandProxy)

	v := &Tesla{
		embed:      &cc.embed,
		Provider:   tesla.NewProvider(vehicle, cc.Cache),
		Controller: tesla.NewController(vehicle.WithClient(tcc)),
	}

	v.fromVehicle(vehicle.DisplayName, 0)

	return v, nil
}
