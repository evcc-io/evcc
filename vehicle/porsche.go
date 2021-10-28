package vehicle

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/porsche"
)

type porscheProvider interface {
	api.Battery
	api.ChargeState
	api.VehicleRange
	api.VehicleClimater
	api.VehicleFinishTimer
	api.VehicleOdometer
}

// Porsche is an api.Vehicle implementation for Porsche cars
type Porsche struct {
	*Embed
	porscheProvider // provides the api implementations
}

func init() {
	registry.Add("porsche", NewPorscheFromConfig, defaults())
}

// NewPorscheFromConfig creates a new vehicle
func NewPorscheFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := defaults()

	log := util.NewLogger("porsche")

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	identity := porsche.NewIdentity(log, cc.User, cc.Password)

	accessTokens, err := identity.Login()
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	vin, err := identity.FindVehicle(accessTokens, cc.VIN)
	if err != nil {
		return nil, err
	}

	provider := porsche.NewProvider(log, identity, accessTokens, vin, cc.Cache)

	v := &Porsche{
		Embed:           &cc.Embed,
		porscheProvider: provider,
	}

	return v, err
}
