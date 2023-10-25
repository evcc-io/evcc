package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/ford"
	"github.com/evcc-io/evcc/vehicle/ford/autonomic"
)

// https://github.com/d4v3y0rk/ffpass-module
// https://github.com/ianjwhite99/connected-car-node-sdk
// https://github.com/TA2k/ioBroker.ford

// Ford is an api.Vehicle implementation for Ford cars
type Ford struct {
	*embed
	*ford.Provider
}

func init() {
	registry.Add("ford", NewFordFromConfig)
}

// NewFordFromConfig creates a new vehicle
func NewFordFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	v := &Ford{
		embed: &cc.embed,
	}

	log := util.NewLogger("ford").Redact(cc.User, cc.Password, cc.VIN)
	identity := ford.NewIdentity(log, cc.User, cc.Password)

	err := identity.Login()
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	api := ford.NewAPI(log, identity)

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)
	if err != nil {
		return nil, err
	}

	autoIdentity, err := autonomic.NewIdentity(log, identity)
	if err != nil {
		return nil, err
	}

	autoApi := autonomic.NewAPI(log, autoIdentity)

	if err == nil {
		v.Provider = ford.NewProvider(autoApi, cc.VIN, cc.Cache)
	}

	return v, err
}
