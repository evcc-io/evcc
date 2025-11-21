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
func NewFordConnectFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed       `mapstructure:",squash"`
		Credentials ClientCredentials
		RedirectURI string
		Tokens_     Tokens // TODO deprecated
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

	log := util.NewLogger("ford").Redact(cc.VIN)
	ts, err := connect.NewIdentity(cc.Credentials.ID, cc.Credentials.Secret, cc.RedirectURI)
	if err != nil {
		return nil, err
	}

	api := connect.NewAPI(log, ts)

	vehicle, err := ensureVehicleEx(cc.VIN, api.Vehicles, func(v connect.Vehicle) (string, error) {
		return api.VIN(v.VehicleID)
	})

	if err == nil {
		v.fromVehicle(vehicle.NickName, 0)
		v.Provider = connect.NewProvider(api, vehicle.VehicleID, cc.Cache)
	}

	return v, err
}
