package vehicle

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/vehicle/porsche"
	"github.com/andig/evcc/util"
)

type porscheProvider interface {
	api.Battery
	api.VehicleRange
}

// Porsche is an api.Vehicle implementation for Porsche cars
type Porsche struct {
	*embed
	porscheProvider // provides the api implementations
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

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	identity := porsche.NewIdentity(log, cc.User, cc.Password)

	accessTokens, err := identity.Login()
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	vehicle, err := identity.FindVehicle(accessTokens, cc.VIN)
	if err != nil {
		return nil, err
	}

	var provider porscheProvider
	if vehicle.EmobilityVehicle {
		provider = porsche.NewEMobilityProvider(log, identity, accessTokens.EmobilityToken, vehicle.VIN, cc.Cache)
	} else {
		provider = porsche.NewProvider(log, identity, accessTokens.Token, vehicle.VIN, cc.Cache)
	}

	v := &Porsche{
		embed:           &embed{cc.Title, cc.Capacity},
		porscheProvider: provider,
	}

	return v, err
}
