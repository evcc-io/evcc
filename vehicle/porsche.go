package vehicle

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/porsche"
)

// Porsche is an api.Vehicle implementation for Porsche cars using the Porsche
// Connect (PPA) app backend. Login happens in the ui via evcc's provider auth
// (see vehicle/porsche).
type Porsche struct {
	*embed
	*porsche.Provider
}

func init() {
	registry.AddCtx("porsche", NewPorscheFromConfig)
}

// NewPorscheFromConfig creates a new vehicle
func NewPorscheFromConfig(_ context.Context, other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		Cache          time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("porsche").Redact(cc.VIN)

	// no api call here: the vehicle exists before the account is connected,
	// the vin is resolved from the account on first use if not configured
	identity, err := porsche.NewIdentity(log, cc.User, cc.Password)
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
