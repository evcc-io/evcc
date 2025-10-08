package vehicle

import (
	"context"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/psa"
)

// https://github.com/TA2k/ioBroker.psa

func init() {
	registry.AddCtx("citroen", func(ctx context.Context, other map[string]any) (api.Vehicle, error) {
		return newPSA(ctx, "citroen", "clientsB2CCitroen", other)
	})
	registry.AddCtx("ds", func(ctx context.Context, other map[string]any) (api.Vehicle, error) {
		return newPSA(ctx, "ds", "clientsB2CDS", other)
	})
	registry.AddCtx("opel", func(ctx context.Context, other map[string]any) (api.Vehicle, error) {
		return newPSA(ctx, "opel", "clientsB2COpel", other)
	})
	registry.AddCtx("peugeot", func(ctx context.Context, other map[string]any) (api.Vehicle, error) {
		return newPSA(ctx, "peugeot", "clientsB2CPeugeot", other)
	})
}

// PSA is an api.Vehicle implementation for PSA cars
type PSA struct {
	*embed
	*psa.Provider // provides the api implementations
}

// newPSA creates a new vehicle
func newPSA(ctx context.Context, brand, realm string, other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed    `mapstructure:",squash"`
		VIN      string
		User     string
		Password string `mapstructure:"password"`
		Country  string
		Tokens   Tokens
		Cache    time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" {
		return nil, api.ErrMissingCredentials
	}

	token, err := cc.Tokens.Token()
	if err != nil {
		return nil, err
	}

	v := &PSA{
		embed: cc.embed.withContext(ctx),
	}

	log := util.NewLogger(brand)
	log.Redact(cc.User, cc.Tokens.Access, cc.Tokens.Refresh)

	oc := psa.Oauth2Config(brand, strings.ToLower(cc.Country))
	identity, err := psa.NewIdentity(log, brand, cc.User, oc, token)
	if err != nil {
		return nil, err
	}

	// TODO still needed?
	api := psa.NewAPI(log, identity, realm, oc.ClientID)

	vehicle, err := ensureVehicleEx(
		cc.VIN, api.Vehicles,
		func(v psa.Vehicle) (string, error) {
			return v.VIN, nil
		},
	)
	if err != nil {
		return nil, err
	}

	v.Provider = psa.NewProvider(api, vehicle.ID, cc.Cache)

	return v, err
}
