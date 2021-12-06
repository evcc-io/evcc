package vehicle

import (
	"fmt"
	"time"

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
	*embed
	porscheProvider // provides the api implementations
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
		embed:           &cc.embed,
		porscheProvider: provider,
	}

	return v, err
}
