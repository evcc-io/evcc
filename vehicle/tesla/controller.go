package tesla

import (
	"errors"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/evcc-io/tesla-proxy-client"
)

const ProxyBaseUrl = "https://tesla.evcc.io"

type Controller struct {
	vehicle *tesla.Vehicle
	dataG   provider.Cacheable[*tesla.VehicleData]
}

// NewController creates a vehicle current and charge controller
func NewController(vehicle *tesla.Vehicle) *Controller {
	impl := &Controller{
		vehicle: vehicle,
		dataG: provider.ResettableCached(func() (*tesla.VehicleData, error) {
			res, err := vehicle.Data()
			return res, apiError(err)
		}, time.Minute),
	}

	return impl
}

var _ api.CurrentController = (*Controller)(nil)

// MaxCurrent implements the api.CurrentController interface
func (v *Controller) MaxCurrent(current int64) error {
	if !sponsor.IsAuthorized() {
		return api.ErrSponsorRequired
	}

	v.dataG.Reset()

	return apiError(v.vehicle.SetChargingAmps(int(current)))
}

var _ api.CurrentGetter = (*Controller)(nil)

// StartCharge implements the api.VehicleChargeController interface
func (v *Controller) GetMaxCurrent() (float64, error) {
	res, err := v.dataG.Get()
	return float64(res.Response.ChargeState.ChargeRate), err
}

var _ api.ChargeController = (*Controller)(nil)

// ChargeEnable implements the api.ChargeController interface
func (v *Controller) ChargeEnable(enable bool) error {
	if !sponsor.IsAuthorized() {
		return api.ErrSponsorRequired
	}

	var err error

	if enable {
		err = apiError(v.vehicle.StartCharging())
		if err != nil && slices.Contains([]string{"complete", "is_charging"}, err.Error()) {
			return nil
		}
	} else {
		err = apiError(v.vehicle.StopCharging())

		// ignore sleeping vehicle
		if errors.Is(err, api.ErrAsleep) {
			err = nil
		}
	}

	return err
}
