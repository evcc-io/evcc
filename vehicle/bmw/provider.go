package bmw

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

// Provider implements the evcc vehicle api
type Provider struct {
	statusG func() (interface{}, error)
}

// NewProvider provides the evcc vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.NewCached(func() (interface{}, error) {
			return api.Status(vin)
		}, cache).InterfaceGetter(),
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(VehicleStatus); err == nil && ok {
		return float64(res.Properties.ChargingState.ChargePercentage), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if res, ok := res.(VehicleStatus); err == nil && ok {
		if res.Properties.ChargingState.IsChargerConnected {
			status = api.StatusB
		}
		if res.Properties.ChargingState.State == "CHARGING" {
			status = api.StatusC
		}
	}

	return status, err
}

// var _ api.VehicleFinishTimer = (*Provider)(nil)

// // FinishTime implements the api.VehicleFinishTimer interface
// func (v *Provider) FinishTime() (time.Time, error) {
// 	res, err := v.statusG()
// 	if res, ok := res.(StatusResponse); err == nil && ok {
// 		ctr := res.VehicleStatus.ChargingTimeRemaining
// 		return time.Now().Add(time.Duration(ctr) * time.Minute), err
// 	}

// 	return time.Time{}, err
// }

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if res, ok := res.(VehicleStatus); err == nil && ok {
		return int64(res.Properties.ElectricRange.Distance.Value), nil
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(VehicleStatus); err == nil && ok {
		return float64(res.Status.CurrentMileage.Mileage), nil
	}

	return 0, err
}
