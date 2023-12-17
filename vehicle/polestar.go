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
		Cache          time.Duration
		Timeout        time.Duration
	}{
		Cache:   interval,
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("polestar").Redact(cc.User, cc.Password, cc.VIN)

	v := &Polestar{
		embed: &cc.embed,
	}

	identity := polestar.NewIdentity(log)
	if err := identity.Login(cc.User, cc.Password); err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := polestar.NewAPI(log, identity)

	vehicle, err := ensureVehicleEx(cc.VIN, func() ([]polestar.ConsumerCar, error) {
		ctx, cancel := context.WithTimeout(context.Background(), cc.Timeout)
		defer cancel()
		return api.Vehicles(ctx)
	}, func(v polestar.ConsumerCar) string {
		return v.VIN
	})

	if err == nil {
		v.Provider = polestar.NewProvider(log, api, vehicle.VIN, cc.Timeout, cc.Cache)
	}

	return v, err
}
