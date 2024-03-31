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
	current int64
	dataG   provider.Cacheable[float64]
}

// NewController creates a vehicle current and charge controller
func NewController(ro, rw *tesla.Vehicle) *Controller {
	v := &Controller{
		vehicle: rw,
	}

	v.dataG = provider.ResettableCached(func() (float64, error) {
		if v.current >= 6 {
			// assume match above 6A to save API requests
			return float64(v.current), nil
		}
		res, err := ro.Data()
		return float64(res.Response.ChargeState.ChargeAmps), apiError(err)
	}, time.Minute)

	return v
}

var _ api.CurrentController = (*Controller)(nil)

// MaxCurrent implements the api.CurrentController interface
func (v *Controller) MaxCurrent(current int64) error {
	if !sponsor.IsAuthorized() {
		return api.ErrSponsorRequired
	}

	v.current = current
	v.dataG.Reset()

	return apiError(v.vehicle.SetChargingAmps(int(current)))
}

var _ api.CurrentGetter = (*Controller)(nil)

// StartCharge implements the api.VehicleChargeController interface
func (v *Controller) GetMaxCurrent() (float64, error) {
	return v.dataG.Get()
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
