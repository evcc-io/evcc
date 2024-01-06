package vehicle

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/porsche"
)

// Porsche is an api.Vehicle implementation for Porsche cars
type Porsche struct {
	*embed
	*porsche.Provider
}

func init() {
	registry.Add("porsche", NewPorscheFromConfig)
}

// NewPorscheFromConfig creates a new vehicle
func NewPorscheFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	log := util.NewLogger("porsche").Redact(cc.User, cc.Password, cc.VIN)
	ts, err := porsche.NewIdentity(log, cc.User, cc.Password)
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	api := porsche.NewAPI(log, ts)

	vehicle, err := ensureVehicleEx(
		cc.VIN, api.Vehicles,
		func(v porsche.Vehicle) string {
			return v.VIN
		},
	)
	if err != nil {
		return nil, err
	}

	// check if vehicle is paired
	if res, err := api.PairingStatus(vehicle.VIN); err == nil && !porsche.IsPaired(res.Status) {
		return nil, errors.New("vehicle is not paired with the My Porsche account")
	}

	emobApi := porsche.NewEmobilityAPI(log, ts)
	capabilities, err := emobApi.Capabilities(vehicle.VIN)
	if err != nil {
		return nil, err
	}

	provider := porsche.NewProvider(log, api, emobApi, vehicle.VIN, capabilities.CarModel, cc.Cache)

	v := &Porsche{
		embed:    &cc.embed,
		Provider: provider,
	}

	return v, err
}
