package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/subaru"
)

// Subaru is an api.Vehicle implementation for Subaru cars
type Subaru struct {
	*embed
	*subaru.Provider
}

func init() {
	registry.Add("subaru", NewSubaruFromConfig)
}

// NewSubaruFromConfig creates a new vehicle
func NewSubaruFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	v := &Subaru{
		embed: &cc.embed,
	}

	log := util.NewLogger("subaru").Redact(cc.User, cc.Password, cc.VIN)
	identity := subaru.NewIdentity(log)

	err := identity.Login(cc.User, cc.Password)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := subaru.NewAPI(log, identity)

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)
	if err == nil {
		v.Provider = subaru.NewProvider(api, cc.VIN, cc.Cache)
	}

	return v, err
}
