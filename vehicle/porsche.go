package vehicle

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/porsche"
	"golang.org/x/oauth2"
)

// Porsche is an api.Vehicle implementation for Porsche cars using the Porsche
// Connect (PPA) app backend. Authentication happens in the browser via evcc's
// provider-auth (see vehicle/porsche).
type Porsche struct {
	*embed
	*porsche.Provider
}

func init() {
	registry.AddCtx("porsche", NewPorscheFromConfig)
}

// NewPorscheFromConfig creates a new vehicle
func NewPorscheFromConfig(ctx context.Context, other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed  `mapstructure:",squash"`
		Tokens Tokens
		VIN    string
		Cache  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// VIN is optional: the vehicle is built without an API call (so the auth
	// provider registers and the UI shows the login button before login). If no
	// VIN is configured it is resolved from the account on first use.
	log := util.NewLogger("porsche").Redact(cc.VIN, cc.Tokens.Access, cc.Tokens.Refresh)

	// optional seed token from `evcc token` / config (web login is the default)
	var seed *oauth2.Token
	if token, err := cc.Tokens.Token(); err == nil {
		seed = token
	}

	identity, err := porsche.NewIdentity(ctx, log, seed)
	if err != nil {
		return nil, err
	}

	api := porsche.NewAPI(log, identity)

	v := &Porsche{
		embed:    &cc.embed,
		Provider: porsche.NewProvider(api, cc.VIN, cc.Cache),
	}

	return v, nil
}
