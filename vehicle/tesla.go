package vehicle

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/bogosj/tesla"
	"golang.org/x/oauth2"
)

// Tesla is an api.Vehicle implementation for Tesla cars
type Tesla struct {
	*embed
	vehicle        *tesla.Vehicle
	chargeStateG   func() (float64, error)
	chargedEnergyG func() (float64, error)
}

// teslaTokens contains access and refresh tokens
type teslaTokens struct {
	Access, Refresh string
}

func init() {
	registry.Add("tesla", NewTeslaFromConfig)
}

// NewTeslaFromConfig creates a new Tesla vehicle
func NewTeslaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title                  string
		Capacity               int64
		ClientID, ClientSecret string
		User, Password         string
		Tokens                 teslaTokens
		VIN                    string
		Cache                  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" && cc.Tokens.Access == "" {
		return nil, errors.New("missing credentials")
	}

	v := &Tesla{
		embed: &embed{cc.Title, cc.Capacity},
	}

	// authenticated http client with logging injected to the Tesla client
	log := util.NewLogger("tesla")
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewHelper(log).Client)

	var options []tesla.ClientOption
	if cc.Tokens.Access != "" {
		options = append(options, tesla.WithToken(&oauth2.Token{
			AccessToken:  cc.Tokens.Access,
			RefreshToken: cc.Tokens.Refresh,
			Expiry:       time.Now(),
		}))
	} else {
		options = append(options, tesla.WithCredentials(cc.User, cc.Password))
	}

	client, err := tesla.NewClient(ctx, options...)
	if err != nil {
		return nil, err
	}

	vehicles, err := client.Vehicles()
	if err != nil {
		return nil, err
	}

	if cc.VIN == "" && len(vehicles) == 1 {
		v.vehicle = vehicles[0]
	} else {
		for _, vehicle := range vehicles {
			if vehicle.Vin == strings.ToUpper(cc.VIN) {
				v.vehicle = vehicle
			}
		}
	}

	if v.vehicle == nil {
		return nil, errors.New("vin not found")
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()
	v.chargedEnergyG = provider.NewCached(v.chargedEnergy, cc.Cache).FloatGetter()

	return v, nil
}

// chargeState implements the api.Vehicle interface
func (v *Tesla) chargeState() (float64, error) {
	state, err := v.vehicle.ChargeState()
	if err != nil {
		return 0, err
	}
	return float64(state.BatteryLevel), nil
}

// SoC implements the api.Vehicle interface
func (v *Tesla) SoC() (float64, error) {
	return v.chargeStateG()
}

// chargedEnergy implements the ChargeRater.ChargedEnergy interface
func (v *Tesla) chargedEnergy() (float64, error) {
	state, err := v.vehicle.ChargeState()
	if err != nil {
		return 0, err
	}
	return state.ChargeEnergyAdded, nil
}

// ChargedEnergy implements the ChargeRater.ChargedEnergy interface
func (v *Tesla) ChargedEnergy() (float64, error) {
	return v.chargedEnergyG()
}
