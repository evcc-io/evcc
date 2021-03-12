package vehicle

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/mercedes"
	"golang.org/x/oauth2"
)

// Mercedes is an api.Vehicle implementation for Mercedes cars
type Mercedes struct {
	*embed
	oc    *oauth2.Config
	token *oauth2.Token
}

func init() {
	registry.Add("mercedes", NewMercedesFromConfig)
}

// NewMercedesFromConfig creates a new Mercedes vehicle
func NewMercedesFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title                  string
		Capacity               int64
		ClientID, ClientSecret string
		User, Password         string
		Tokens                 Tokens
		VIN                    string
		Cache                  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientID == "" && cc.Tokens.Access == "" {
		return nil, errors.New("missing credentials")
	}

	v := &Mercedes{
		embed: &embed{cc.Title, cc.Capacity},
	}

	log := util.NewLogger("mercedes")

	identity := mercedes.NewIdentity(cc.ClientID, cc.ClientSecret)
	if err := identity.Login(log); err != nil {
		return nil, err
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewHelper(log).Client)
	client := identity.AuthConfig.Client(ctx, identity.Token())

	client = request.NewHelper(log).Client
	uri := fmt.Sprintf("https://api.mercedes-benz.com/vehicledata_tryout/v2/vehicles/%s/containers/electricvehicle", "WDB111111ZZZ22222")
	client.Get(uri)
	// authenticated http client with logging injected to the Mercedes client

	// vehicles, err := client.Vehicles()
	// if err != nil {
	// 	return nil, err
	// }

	// if cc.VIN == "" && len(vehicles) == 1 {
	// 	v.vehicle = vehicles[0]
	// } else {
	// 	for _, vehicle := range vehicles {
	// 		if vehicle.Vin == strings.ToUpper(cc.VIN) {
	// 			v.vehicle = vehicle
	// 		}
	// 	}
	// }

	return v, nil
}

// chargeState implements the api.Vehicle interface
func (v *Mercedes) chargeState() (float64, error) {
	// state, err := v.vehicle.ChargeState()
	// if err != nil {
	// 	return 0, err
	// }
	// return float64(state.BatteryLevel), nil
	return 0, nil
}

// SoC implements the api.Vehicle interface
func (v *Mercedes) SoC() (float64, error) {
	// return v.chargeStateG()
	return 0, nil
}
