package vehicle

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/porsche"
	"github.com/thoas/go-funk"
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
	identity := porsche.NewIdentity(log)

	if err := identity.Login(cc.User, cc.Password); err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	api := porsche.NewAPI(log, identity.DefaultSource)

	if cc.VIN == "" {
		vehicles, err := api.Vehicles()
		if err == nil {
			cc.VIN, err = findVehicle(funk.Map(vehicles, func(v porsche.Vehicle) string {
				return v.VIN
			}).([]string), nil)
		}

		if err != nil {
			return nil, err
		}

		log.DEBUG.Printf("found vehicle: %v", cc.VIN)
	}

	// check if vehicle is paired
	if res, err := api.PairingStatus(cc.VIN); err == nil && res.Status != porsche.PairingComplete {
		return nil, errors.New("vehicle is not paired with the My Porsche account")
	}

	// check if vehicle provides status:
	// some PHEVs do not provide any data
	if _, err := api.Status(cc.VIN); err != nil {
		return nil, errors.New("vehicle is not capable of providing data")
	}

	// get eMobility capabilities

	// Note: As of 27.10.21 the capabilities API needs to be called AFTER a
	//   call to status() as it otherwise returns an HTTP 502 error.
	//   The reason is unknown, even when tested with 100% identical Headers.
	//   It seems to be a new backend related issue.
	if _, err := api.Status(cc.VIN); err != nil {
		return nil, err
	}

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
