package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/myuconnect"
)

// https://github.com/TA2k/ioBroker.fiat

// Fiat is an api.Vehicle implementation for Fiat cars
type Fiat struct {
	*embed
	*myuconnect.Provider
}

func init() {
	registry.Add("fiat", func(other map[string]interface{}) (api.Vehicle, error) {
		return NewFiatJeepFromConfig(myuconnect.Fiat, other)
	})
	registry.Add("jeep", func(other map[string]interface{}) (api.Vehicle, error) {
		return NewFiatJeepFromConfig(myuconnect.Jeep, other)
	})
}

// NewFiatJeepFromConfig creates a new vehicle
func NewFiatJeepFromConfig(params myuconnect.Params, other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed                    `mapstructure:",squash"`
		User, Password, VIN, PIN string
		Expiry                   time.Duration
		Cache                    time.Duration
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

	v := &Fiat{
		embed: &cc.embed,
	}

	log := util.NewLogger("fiat").Redact(cc.User, cc.Password, cc.VIN)
	identity := myuconnect.NewIdentity(log, params, cc.User, cc.Password)

	err := identity.Login()
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	api := myuconnect.NewAPI(log, params, identity)

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		v.Provider = myuconnect.NewProvider(api, cc.VIN, cc.PIN, cc.Expiry, cc.Cache)
	}

	return v, err
}
