package vehicle

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	mfa "github.com/andig/evcc/vehicle/tesla"
	"github.com/jsgoecke/tesla"
)

// Tesla is an api.Vehicle implementation for Tesla cars
type Tesla struct {
	*embed
	vehicle        *tesla.Vehicle
	chargeStateG   func() (float64, error)
	chargedEnergyG func() (float64, error)
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
		Token                  string
		VIN                    string
		Cache                  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Tesla{
		embed: &embed{cc.Title, cc.Capacity},
	}

	auth := &tesla.Auth{
		Email:    cc.User,
		Password: cc.Password,
	}

	var err error
	if cc.Token != "" {
		cc.Token, err = v.token(auth)
	}

	var client *tesla.Client
	if err == nil {
		client, err = tesla.NewClientWithToken(auth, &tesla.Token{
			AccessToken: cc.Token,
			TokenType:   "Bearer",
			Expires:     time.Now().Add(45 * 24 * time.Hour).Unix(),
		})
	}

	if err != nil {
		return nil, err
	}

	vehicles, err := client.Vehicles()
	if err != nil {
		return nil, err
	}

	if cc.VIN == "" && len(vehicles) == 1 {
		v.vehicle = vehicles[0].Vehicle
	} else {
		for _, vehicle := range vehicles {
			if vehicle.Vin == strings.ToUpper(cc.VIN) {
				v.vehicle = vehicle.Vehicle
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

// token creates the Tesla access token
func (v *Tesla) token(auth *tesla.Auth) (string, error) {
	ctx := context.Background()
	client, err := mfa.NewClient(util.NewLogger("tesla"))
	if err != nil {
		return "", err
	}

	cb := func() (string, error) {
		return "", errors.New("multi-factor authentication not implemented, use `evcc tesla-token` to create an access token and add `token` to vehicle config")
	}

	return client.Login(ctx, auth.Email, auth.Password, cb)
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
