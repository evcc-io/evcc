package vehicle

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/fiat"
)

// https://github.com/TA2k/ioBroker.fiat

// Fiat is an api.Vehicle implementation for Fiat cars
type Fiat struct {
	*embed
	*fiat.Provider
	*fiat.Controller
}

func init() {
	registry.AddCtx("fiat", NewFiatFromConfig)
}

// NewFiatFromConfig creates a new vehicle
func NewFiatFromConfig(ctx context.Context, other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed                    `mapstructure:",squash"`
		User, Password, VIN, PIN string
		Expiry                   time.Duration
		Cache                    time.Duration
	}{
		Expiry: expiry,
		Cache:  interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	v := &Fiat{
		embed: &cc.embed,
	}

	log := util.NewLogger("fiat").Redact(cc.User, cc.Password, cc.VIN)
	identity := fiat.NewIdentity(log, ctx, cc.User, cc.Password)

	err := identity.Login()
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	api := fiat.NewAPI(log, identity)

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		v.Provider = fiat.NewProvider(api, cc.VIN, cc.PIN, cc.Expiry, cc.Cache)
		v.Controller = fiat.NewController(v.Provider, api, log, cc.VIN, cc.PIN)
	}

	return v, err
}
