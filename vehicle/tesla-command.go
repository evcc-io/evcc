package vehicle

import (
	"context"
	"os"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	vc "github.com/evcc-io/evcc/vehicle/tesla-vehicle-command"
	"golang.org/x/oauth2"
)

// TeslaCommand is an api.Vehicle implementation for Tesla cars using the official Tesla vehicle-command api.
type TeslaCommand struct {
	*embed
	*vc.Provider
}

func init() {
	if id := os.Getenv("TESLA_CLIENT_ID"); id != "" {
		vc.OAuth2Config.ClientID = id
	}
	if secret := os.Getenv("TESLA_CLIENT_SECRET"); secret != "" {
		vc.OAuth2Config.ClientSecret = secret
	}
	if vc.OAuth2Config.ClientID != "" {
		registry.Add("tesla-command", NewTeslaCommandFromConfig)
	}
}

// const privateKeyFile = "tesla-privatekey.pem"

// NewTeslaCommandFromConfig creates a new vehicle
func NewTeslaCommandFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	log := util.NewLogger("tesla-command").Redact(
		cc.Tokens.Access, cc.Tokens.Refresh,
		vc.OAuth2Config.ClientID, vc.OAuth2Config.ClientSecret,
	)

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewClient(log))
	ts := vc.OAuth2Config.TokenSource(ctx, token)

	identity, err := vc.NewIdentity(log, ts)
	if err != nil {
		return nil, err
	}

	api := vc.NewAPI(log, identity, cc.Timeout)

	// validate base url
	if _, err := api.Region(); err != nil {
		return nil, err
	}

	vehicle, err := ensureVehicleEx(
		cc.VIN, api.Vehicles,
		func(v *vc.Vehicle) string {
			return v.Vin
		},
	)
	if err != nil {
		return nil, err
	}

	v := &TeslaCommand{
		embed:    &cc.embed,
		Provider: vc.NewProvider(api, vehicle.ID, cc.Cache),
	}

	if v.Title_ == "" {
		v.Title_ = vehicle.DisplayName
	}
	/*
		privKey, err := protocol.LoadPrivateKey(privateKeyFile)
		if err != nil {
			log.WARN.Println("private key not found, commands are disabled")
			return v, nil
		}

		vv, err := identity.Account().GetVehicle(context.Background(), vehicle.Vin, privKey, cache.New(8))
		if err != nil {
			return nil, err
		}

		cs, err := vc.NewCommandSession(vv, cc.Timeout)
		if err != nil {
			return nil, err
		}

		res := &struct {
			*TeslaCommand
			*vc.CommandSession
		}{
			TeslaCommand:   v,
			CommandSession: cs,
		}
	*/

	return v, nil
}
