package vehicle

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle/porsche"
	"github.com/andig/evcc/util"
)

type SoCAndRange interface {
	api.Battery
	api.VehicleRange
}

// Porsche is an api.Vehicle implementation for Porsche cars
type Porsche struct {
	*embed
	SoCAndRange // provides the api implementations
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

	api := porsche.NewAPI(log, cc.User, cc.Password)
	err = api.Login()
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	vehicle, err := api.FindVehicle(cc.VIN)
	if err != nil {
		return nil, err
	}

	var provider SoCAndRange
	if vehicle.EmobilityVehicle {
		provider = porsche.NewEMobilityProvider(api, vehicle.VIN, cc.Cache)
	} else {
		provider = porsche.NewProvider(api, vehicle.VIN, cc.Cache)
	}

	v.SoCAndRange = provider

	return v, err
}
