package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/mb"
	"github.com/evcc-io/evcc/vehicle/smart"
)

// Smart is an api.Vehicle implementation for Smart cars
type Smart struct {
	*embed
	*smart.Provider
}

func init() {
	registry.Add("smart", NewSmartFromConfig)
}

// NewSmartFromConfig creates a new vehicle
func NewSmartFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		Expiry         time.Duration
		Cache          time.Duration
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

	log := util.NewLogger("smart").Redact(cc.User, cc.Password, cc.VIN)

	v := &Smart{
		embed: &cc.embed,
	}

	identity := mb.NewIdentity(log, smart.OAuth2Config)
	err := identity.Login(cc.User, cc.Password)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := smart.NewAPI(log, identity)

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		v.Provider = smart.NewProvider(log, api, cc.VIN, cc.Expiry, cc.Cache)
	}

	return v, err
}
