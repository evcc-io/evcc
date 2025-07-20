package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/toyota"
)

// Toyota is an api.Vehicle implementation for Toyota cars
type Toyota struct {
	*embed
	*toyota.Provider
}

func init() {
	registry.Add("toyota", NewToyotaFromConfig)
}

// NewToyotaFromConfig creates a new vehicle
func NewToyotaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	v := &Toyota{
		embed: &cc.embed,
	}

	log := util.NewLogger("toyota").Redact(cc.User, cc.Password, cc.VIN)
	identity := toyota.NewIdentity(log)

	err := identity.Login(cc.User, cc.Password)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := toyota.NewAPI(log, identity)

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)
	if err == nil {
		v.Provider = toyota.NewProvider(api, cc.VIN, cc.Cache)
	}

	return v, err
}
