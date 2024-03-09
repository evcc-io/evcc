package tesla

import (
	"errors"
	"slices"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/evcc-io/tesla-proxy-client"
)

const ProxyBaseUrl = "https://tesla.evcc.io"

type Controller struct {
	vehicle *tesla.Vehicle
}

// NewController creates a vehicle current and charge controller
func NewController(vehicle *tesla.Vehicle) *Controller {
	impl := &Controller{
		vehicle: vehicle,
	}
	return impl
}

var _ api.CurrentController = (*Controller)(nil)

// StartCharge implements the api.VehicleChargeController interface
func (v *Controller) MaxCurrent(current int64) error {
	if !sponsor.IsAuthorized() {
		return api.ErrSponsorRequired
	}

	return apiError(v.vehicle.SetChargingAmps(int(current)))
}

var _ api.VehicleChargeController = (*Controller)(nil)

// StartCharge implements the api.VehicleChargeController interface
func (v *Controller) StartCharge() error {
	if !sponsor.IsAuthorized() {
		return api.ErrSponsorRequired
	}

	err := apiError(v.vehicle.StartCharging())
	if err != nil && slices.Contains([]string{"complete", "is_charging"}, err.Error()) {
		return nil
	}
	return err
}

// StopCharge implements the api.VehicleChargeController interface
func (v *Controller) StopCharge() error {
	if !sponsor.IsAuthorized() {
		return api.ErrSponsorRequired
	}

	err := apiError(v.vehicle.StopCharging())

	// ignore sleeping vehicle
	if errors.Is(err, api.ErrAsleep) {
		err = nil
	}

	return err
}
