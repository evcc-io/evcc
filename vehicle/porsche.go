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
	identity := porsche.NewIdentity(log, cc.User, cc.Password)

	err := identity.Login()
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	api := porsche.NewAPI(log, identity.DefaultSource)
	mobile := porsche.NewMobileAPI(log, identity.MobileSource)

	cc.VIN, err = ensureVehicle(cc.VIN, func() ([]string, error) {
		mobileVehicles, err := mobile.Vehicles()
		if err == nil {
			return funk.Map(mobileVehicles, func(v porsche.StatusResponseMobile) string {
				return v.VIN
			}).([]string), err
		}

		vehicles, err := api.Vehicles()
		return funk.Map(vehicles, func(v porsche.Vehicle) string {
			return v.VIN
		}).([]string), err
	})

	if err != nil {
		return nil, err
	}

	// check if vehicle is paired
	if res, err := api.PairingStatus(cc.VIN); err == nil && res.Status != porsche.PairingComplete {
		return nil, errors.New("vehicle is not paired with the My Porsche account")
	}

	// check if vehicle provides status:
	// some PHEVs do not provide any data
	statusAvailable := false
	if _, err := mobile.Status(cc.VIN); err == nil {
		statusAvailable = true
	}

	if _, err := api.Status(cc.VIN); err == nil {
		statusAvailable = true
	}

	if !statusAvailable {
		return nil, errors.New("vehicle is not capable of providing data")
	}

	// get eMobility capabilities

	// Note: As of 27.10.21 the capabilities API needs to be called AFTER a
	//   call to status() as it otherwise returns an HTTP 502 error.
	//   The reason is unknown, even when tested with 100% identical Headers.
	//   It seems to be a new backend related issue.
	var emobility *porsche.EmobilityAPI
	var capabilities porsche.CapabilitiesResponse
	if _, err := api.Status(cc.VIN); err == nil {
		emobility = porsche.NewEmobilityAPI(log, identity.EmobilitySource)
		capabilities, _ = emobility.Capabilities(cc.VIN)
	}

	provider := porsche.NewProvider(log, api, emobility, mobile, cc.VIN, capabilities.CarModel, cc.Cache)

	v := &Porsche{
		embed:    &cc.embed,
		Provider: provider,
	}

	return v, err
}
