package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/vw/eudataact"
)

// https://github.com/TA2k/ioBroker.vw-connect (EU Data Act data source)

// DriveSomethingGreater is an api.Vehicle implementation for the VW group
// "EU Data Act" data portal (eu-data-act.drivesomethinggreater.com)
type DriveSomethingGreater struct {
	*embed
	*eudataact.Provider
}

func init() {
	registry.Add("drivesomethinggreater", NewDriveSomethingGreaterFromConfig)
}

// NewDriveSomethingGreaterFromConfig creates a new vehicle
func NewDriveSomethingGreaterFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed                      `mapstructure:",squash"`
		Brand, User, Password, VIN string
		Cache                      time.Duration
	}{
		Brand: "Volkswagen",
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	v := &DriveSomethingGreater{
		embed: &cc.embed,
	}

	log := util.NewLogger("dsg").Redact(cc.User, cc.Password, cc.VIN)

	api, err := eudataact.NewAPI(log, cc.Brand, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	vehicle, err := ensureVehicleEx(
		cc.VIN, api.Vehicles,
		func(v eudataact.Vehicle) (string, error) {
			return v.Vin(), nil
		},
	)

	if err == nil {
		v.fromVehicle(vehicle.Name(), 0)
		v.Provider = eudataact.NewProvider(api, vehicle.Vin(), cc.Cache)
	}

	return v, err
}
