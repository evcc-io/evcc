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
	vehicle      *tesla.Vehicle
	chargeStateG func() (interface{}, error)
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

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).InterfaceGetter()

	return v, nil
}

// chargeState implements the charge state api
func (v *Tesla) chargeState() (interface{}, error) {
	return v.vehicle.ChargeState()
}

// SoC implements the api.Vehicle interface
func (v *Tesla) SoC() (float64, error) {
	res, err := v.chargeStateG()

	if res, ok := res.(*tesla.ChargeState); err == nil && ok {
		return float64(res.BatteryLevel), nil
	}

	return 0, err
}

// ChargedEnergy implements the api.ChargeRater interface
func (v *Tesla) ChargedEnergy() (float64, error) {
	res, err := v.chargeStateG()

	if res, ok := res.(*tesla.ChargeState); err == nil && ok {
		return float64(res.ChargeEnergyAdded), nil
	}

	return 0, err
}

// Range implements the api.VehicleRange interface
func (v *Tesla) Range() (int64, error) {
	res, err := v.chargeStateG()

	if res, ok := res.(*tesla.ChargeState); err == nil && ok {
		// miles to km
		return int64(1.609344 * res.EstBatteryRange), nil
	}

	return 0, err
}

// Status implements the api.ChargeState interface
func (v *Tesla) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected
	res, err := v.chargeStateG()

	if res, ok := res.(*tesla.ChargeState); err == nil && ok {
		if res.ChargingState == "Stopped" || res.ChargingState == "NoPower" || res.ChargingState == "Complete" {
			status = api.StatusB
		}
		if res.ChargingState == "Charging" {
			status = api.StatusC
		}
	}

	return status, err
}

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Tesla) FinishTime() (time.Time, error) {
	res, err := v.chargeStateG()

	if res, ok := res.(*tesla.ChargeState); err == nil && ok {
		t := time.Now()
		return t.Add(time.Duration(res.MinutesToFullCharge) * time.Minute), err
	}

	return time.Time{}, err
}
