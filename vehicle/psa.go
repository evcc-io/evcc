package vehicle

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/psa"
	"golang.org/x/oauth2"
)

// https://github.com/TA2k/ioBroker.psa

func init() {
	registry.AddCtx("citroen", func(ctx context.Context, other map[string]any) (api.Vehicle, error) {
		return newPSA(ctx, "citroen", "clientsB2CCitroen", "Citroën", other)
	})
	registry.AddCtx("ds", func(ctx context.Context, other map[string]any) (api.Vehicle, error) {
		return newPSA(ctx, "ds", "clientsB2CDS", "DS", other)
	})
	registry.AddCtx("opel", func(ctx context.Context, other map[string]any) (api.Vehicle, error) {
		return newPSA(ctx, "opel", "clientsB2COpel", "Opel", other)
	})
	registry.AddCtx("peugeot", func(ctx context.Context, other map[string]any) (api.Vehicle, error) {
		return newPSA(ctx, "peugeot", "clientsB2CPeugeot", "Peugeot", other)
	})
}

// PSA is an api.Vehicle implementation for PSA cars. Authentication happens in
// the browser via evcc's provider-auth (see vehicle/psa).
type PSA struct {
	*embed
	*psa.Provider // provides the api implementations
}

// newPSA creates a new vehicle
func newPSA(ctx context.Context, brand, realm, displayName string, other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed    `mapstructure:",squash"`
		VIN      string
		Tokens   Tokens
		Cache    time.Duration
		User     string // deprecated
		Password string // deprecated
		Country  string // deprecated
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger(brand)

	// optional seed token from a legacy `evcc token` config (web login is the default)
	var seed *oauth2.Token
	if token, err := cc.Tokens.Token(); err == nil {
		seed = token
	}

	oc := psa.Oauth2Config(brand, "")
	identity, err := psa.NewIdentity(ctx, log, brand, realm, displayName, oc, seed)
	if err != nil {
		return nil, err
	}

	api := psa.NewAPI(log, identity, realm, oc.ClientID)

	// resolve the vehicle lazily so the vehicle can be built (and the auth
	// provider registered) before the account is authenticated
	vehicle := func() (string, error) {
		vehicle, err := ensureVehicleEx(cc.VIN, api.Vehicles, func(v psa.Vehicle) (string, error) {
			return v.VIN, nil
		})
		return vehicle.ID, err
	}

	v := &PSA{
		embed:    &cc.embed,
		Provider: psa.NewProvider(api, vehicle, cc.Cache),
	}

	return v, nil
}
