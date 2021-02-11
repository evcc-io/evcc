package vehicle

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	auth "github.com/andig/evcc/vehicle/tesla"
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

	log := util.NewLogger("tesla")
	authClient, err := auth.NewClient(log)
	if err != nil {
		return nil, err
	}

	token, err := teslaToken(authClient, cc.User, cc.Password, cc.Tokens)
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	// authenticated http client with logging
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewHelper(log).Client)
	http := authClient.Config.Client(ctx, token)

	// injected to the Tesla client
	client := &tesla.Client{HTTP: http}
	tesla.ActiveClient = client

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

// teslaToken creates the Tesla OAuth token from given credentials
func teslaToken(auth *auth.Client, user, password string, tokens teslaTokens) (*oauth2.Token, error) {
	// without tokens try to login - will fail if MFA enabled
	if tokens.Access == "" {
		token, err := auth.Login(user, password)
		if err != nil {
			err = fmt.Errorf("%w: if using multi-factor authentication, create tokens using `evcc tesla-token`", err)
		}

		return token, err
	}

	// create tokensource with given tokens
	ctx := context.Background()
	ts := auth.Config.TokenSource(ctx, &oauth2.Token{
		AccessToken:  tokens.Access,
		RefreshToken: tokens.Refresh,
		Expiry:       time.Now(),
	})

	// test the token source
	token, err := ts.Token()
	if err != nil {
		err = fmt.Errorf("%w: token refresh failed, check access and refresh tokens are valid", err)
	}

	return token, err
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
