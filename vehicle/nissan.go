package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/nissan"
)

// Credits to
//   https://github.com/Tobiaswk/dartnissanconnect
//   https://github.com/mitchellrj/kamereon-python
//   https://gitlab.com/tobiaswkjeldsen/carwingsflutter

// OAuth base url
// 	 https://prod.eu.auth.kamereon.org/kauth/oauth2/a-ncb-prod/.well-known/openid-configuration

// Nissan is an api.Vehicle implementation for Nissan cars
type Nissan struct {
	*embed
	*nissan.Provider
}

func init() {
	registry.Add("nissan", NewNissanFromConfig)
}

// NewNissanFromConfig creates a new vehicle
func NewNissanFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Version             string
		Expiry              time.Duration
		Cache               time.Duration
	}{
		Version: "v1", // battery api version: v2 for Ariya
		Expiry:  expiry,
		Cache:   interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	v := &Nissan{
		embed: &cc.embed,
	}

	log := util.NewLogger("nissan").Redact(cc.User, cc.Password, cc.VIN)
	identity := nissan.NewIdentity(log)

	err := identity.Login(cc.User, cc.Password)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := nissan.NewAPI(log, identity)

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		v.Provider = nissan.NewProvider(api, cc.VIN, cc.Version, cc.Expiry, cc.Cache)
	}

	return v, err
}
