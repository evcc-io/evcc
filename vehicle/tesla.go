package vehicle

import (
	"context"
	"os"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
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
	if id := os.Getenv("TESLA_CLIENT_ID"); id != "" {
		tesla.OAuth2Config.ClientID = id
	}
	if secret := os.Getenv("TESLA_CLIENT_SECRET"); secret != "" {
		tesla.OAuth2Config.ClientSecret = secret
	}
	if tesla.OAuth2Config.ClientID != "" {
		registry.Add("tesla", NewTeslaFromConfig)
	}
}

// NewTeslaFromConfig creates a new vehicle
func NewTeslaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed   `mapstructure:",squash"`
		Tokens  Tokens
		VIN     string
		Timeout time.Duration
		Cache   time.Duration
	}{
		Timeout: request.Timeout,
		Cache:   interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	token, err := cc.Tokens.Token()
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("tesla").Redact(
		cc.Tokens.Access, cc.Tokens.Refresh,
		tesla.OAuth2Config.ClientID, tesla.OAuth2Config.ClientSecret,
	)

	identity, err := tesla.NewIdentity(log, token)
	if err != nil {
		return nil, err
	}

	hc := request.NewClient(log)
	hc.Transport = &oauth2.Transport{
		Source: identity,
		Base:   hc.Transport,
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
		func(v *tesla.Vehicle) string {
			return v.Vin
		},
	)
	if err != nil {
		return nil, err
	}

	// proxy client
	pc := request.NewClient(log)
	pc.Transport = &transport.Decorator{
		Decorator: transport.DecorateHeaders(map[string]string{
			"X-Auth-Token": sponsor.Token,
		}),
		Base: hc.Transport,
	}

	tcc, err := teslaclient.NewClient(context.Background(), teslaclient.WithClient(pc))
	if err != nil {
		return nil, err
	}
	tcc.SetBaseUrl(tesla.ProxyBaseUrl)

	v := &Tesla{
		embed:      &cc.embed,
		Provider:   tesla.NewProvider(vehicle, cc.Cache),
		Controller: tesla.NewController(vehicle.WithClient(tcc)),
	}

	if v.Title_ == "" {
		v.Title_ = vehicle.DisplayName
	}

	return v, nil
}
