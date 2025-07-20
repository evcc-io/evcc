package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/ford/connect"
)

// https://developer.ford.com/apis/fordconnect

// FordConnect is an api.Vehicle implementation for Ford cars
type FordConnect struct {
	*embed
	*connect.Provider
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

	api := connect.NewAPI(log, identity)

	vehicle, err := ensureVehicleEx(cc.VIN, api.Vehicles, func(v connect.Vehicle) (string, error) {
		return api.VIN(v.VehicleID)
	})

	if err == nil {
		v.fromVehicle(vehicle.NickName, 0)
		v.Provider = connect.NewProvider(api, vehicle.VehicleID, cc.Cache)
	}

	return v, err
}
