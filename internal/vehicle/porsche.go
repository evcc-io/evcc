package vehicle

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle/porsche"
	"github.com/andig/evcc/util"
)

// Porsche is an api.Vehicle implementation for Porsche cars
type Porsche struct {
	*embed
	api.Battery // provides the api implementations
}

func init() {
	registry.Add("porsche", NewPorscheFromConfig)
}

// NewPorscheFromConfig creates a new vehicle
func NewPorscheFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	log := util.NewLogger("porsche")

	var err error

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Porsche{
		embed: &embed{cc.Title, cc.Capacity},
	}

	porscheAPI := porsche.NewAPI(log, cc.User, cc.Password)
	err = porscheAPI.Login()
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	vehicle, err := porscheAPI.FindVehicle(cc.VIN)
	if err != nil {
		return nil, err
	}

	var provider api.Battery
	if vehicle.EmobilityVehicle {
		provider = porsche.NewEMobilityProvider(porscheAPI, vehicle.VIN, cc.Cache)
	} else {
		provider = porsche.NewProvider(porscheAPI, vehicle.VIN, cc.Cache)
	}

	v.Battery = provider

	return v, err
}
