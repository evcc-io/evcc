package vehicle

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/polestar"
)

// Polestar is an api.Vehicle implementation for Polestar cars
type Polestar struct {
	*embed
	*polestar.Provider
}

func init() {
	registry.Add("polestar", NewPolestarFromConfig)
}

// NewPolestarFromConfig creates a new vehicle
func NewPolestarFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		Timeout        time.Duration
		Expiry         time.Duration
		Cache          time.Duration
	}{
		Timeout: request.Timeout,
		Expiry:  expiry,
		Cache:   interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("polestar").Redact(cc.User, cc.Password, cc.VIN)

	v := &Polestar{
		embed: &cc.embed,
	}

	identity := polestar.NewIdentity(log, polestar.OAuth2Config)
	err := identity.Login(cc.User, cc.Password)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := polestar.NewAPI(log, identity)

	// cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)
	cc.VIN, err = ensureVehicle(cc.VIN, func() ([]string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), cc.Timeout)
		defer cancel()
		return api.Vehicles(ctx)
	})

	if err == nil {
		v.Provider = polestar.NewProvider(log, api, cc.VIN, cc.Expiry, cc.Cache)
	}

	return v, err
}
