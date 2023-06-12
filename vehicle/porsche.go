package vehicle

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/porsche"
	"github.com/samber/lo"
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
	identity := porsche.NewIdentity(log, cc.User, cc.Password)

	err := identity.Login()
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	api := porsche.NewAPI(log, identity.DefaultSource)

	cc.VIN, err = ensureVehicle(cc.VIN, func() ([]string, error) {
		vehicles, err := api.Vehicles()
		return lo.Map(vehicles, func(v porsche.Vehicle, _ int) string {
			return v.VIN
		}), err
	})

	if err != nil {
		return nil, err
	}

	// check if vehicle is paired
	if res, err := api.PairingStatus(cc.VIN); err == nil && !porsche.IsPaired(res.Status) {
		return nil, errors.New("vehicle is not paired with the My Porsche account")
	}

	// get eMobility capabilities
	emobility := porsche.NewEmobilityAPI(log, identity.EmobilitySource)
	capabilities, err := emobility.Capabilities(cc.VIN)
	if err != nil {
		return nil, err
	}

	provider := porsche.NewProvider(log, api, emobility, cc.VIN, capabilities.CarModel, cc.Cache)

	v := &Porsche{
		embed:    &cc.embed,
		Provider: provider,
	}

	return v, err
}
