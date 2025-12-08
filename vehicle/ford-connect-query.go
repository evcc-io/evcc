package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/ford/query"
)

// https://developer.ford.com/apis/fordconnect-query

// FordConnectQuery is an api.Vehicle implementation for Ford cars
type FordConnectQuery struct {
	*embed
	*query.Provider
}

func init() {
	registry.Add("ford-connect-query", NewFordConnectQueryFromConfig)
}

// NewFordConnectQueryFromConfig creates a new vehicle
func NewFordConnectQueryFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed       `mapstructure:",squash"`
		Credentials ClientCredentials
		RedirectURI string
		VIN         string
		Cache       time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &FordConnectQuery{
		embed: &cc.embed,
	}

	if err := cc.Credentials.Error(); err != nil {
		return nil, err
	}

	oc := query.OAuth2Config(cc.Credentials.ID, cc.Credentials.Secret, cc.RedirectURI)
	ts, err := query.NewOAuth(oc, cc.embed.GetTitle())
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("ford").Redact(cc.VIN)
	api := query.NewAPI(log, ts)

	vehicle, err := ensureVehicleEx(cc.VIN, api.Vehicles, func(v query.Vehicle) (string, error) {
		return v.VIN, nil
	})

	if err == nil {
		v.fromVehicle(vehicle.NickName, 0)
		v.Provider = query.NewProvider(api, vehicle.VehicleID, cc.Cache)
	}

	return v, err
}
