package vehicle

import (
	"context"
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/mercedes"
)

// Mercedes is an api.Vehicle implementation for Mercedes-Benz cars
type Mercedes struct {
	*embed
	*mercedes.Provider
}

func init() {
	registry.AddCtx("mercedes", func(ctx context.Context, other map[string]interface{}) (api.Vehicle, error) {
		return newMercedesFromConfig(ctx, "mercedes", other)
	})
	registry.AddCtx("smart-eq", func(ctx context.Context, other map[string]interface{}) (api.Vehicle, error) {
		return newMercedesFromConfig(ctx, "smart-eq", other)
	})
}

// newMercedesFromConfig creates a new vehicle
func newMercedesFromConfig(ctx context.Context, brand string, other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed    `mapstructure:",squash"`
		Tokens   Tokens
		User     string
		Account_ string `mapstructure:"account"` // TODO deprecated
		VIN      string
		Cache    time.Duration
		Region   string
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	token, err := cc.Tokens.Token()
	if err != nil {
		return nil, err
	}

	if cc.User == "" && cc.Account_ != "" {
		cc.User = cc.Account_
	}

	log := util.NewLogger(brand).Redact(cc.Tokens.Access, cc.Tokens.Refresh)
	identity, err := mercedes.NewIdentity(log, token, cc.User, cc.Region)
	if err != nil {
		return nil, err
	}

	api := mercedes.NewAPI(log, identity)

	if brand == "smart-eq" {
		if cc.VIN == "" {
			return nil, errors.New("missing VIN")
		}
	} else {
		cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)
		if err != nil {
			return nil, err
		}
	}

	v := &Mercedes{
		embed:    cc.embed.withContext(ctx),
		Provider: mercedes.NewProvider(api, cc.VIN, cc.Cache),
	}

	return v, err
}
