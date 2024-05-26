package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/ford"
	"github.com/evcc-io/evcc/vehicle/ford/autonomic"
	"github.com/evcc-io/evcc/vehicle/ford/connect"
)

// https://github.com/d4v3y0rk/ffpass-module
// https://github.com/ianjwhite99/connected-car-node-sdk
// https://github.com/TA2k/ioBroker.ford

// FordConnect is an api.Vehicle implementation for Ford cars
type FordConnect struct {
	*embed
	*ford.Provider
}

func init() {
	registry.Add("ford-connect", NewFordConnectFromConfig)
}

// NewFordConnectFromConfig creates a new vehicle
func NewFordConnectFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed       `mapstructure:",squash"`
		Credentials ClientCredentials
		Tokens      Tokens
		VIN         string
		Cache       time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &FordConnect{
		embed: &cc.embed,
	}

	if err := cc.Credentials.Error(); err != nil {
		return nil, err
	}

	token, err := cc.Tokens.Token()
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("ford").Redact(cc.VIN)
	identity := connect.NewIdentity(log, cc.Credentials.ID, cc.Credentials.Secret, token)

	api := ford.NewAPI(log, identity)

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)
	if err != nil {
		return nil, err
	}

	autoIdentity, err := autonomic.NewIdentity(log, identity)
	if err == nil {
		api := autonomic.NewAPI(log, autoIdentity)
		v.Provider = ford.NewProvider(api, cc.VIN, cc.Cache)
	}

	return v, err
}
