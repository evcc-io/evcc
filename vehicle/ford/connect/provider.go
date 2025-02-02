package connect

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

type Provider struct {
	statusG func() (Vehicle, error)
	// refreshG func() error
}

func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.Cached(func() (Vehicle, error) {
			return api.Status(vin)
		}, cache),
		// refreshG: func() error {
		// 	_, err := api.Refresh(vin)
		// 	return err
		// },
	}

	return impl
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	return res.VehicleDetails.BatteryChargeLevel.Value, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	return int64(res.VehicleDetails.BatteryChargeLevel.DistanceToEmpty), err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA

	res, err := v.statusG()
	if err != nil {
		return status, err
	}

	if res.VehicleStatus.PlugStatus.Value {
		status = api.StatusB // plugged

		if res.VehicleStatus.ChargingStatus.Value == "Charging" {
			status = api.StatusC // charging
		}
	}

	return status, nil
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	return res.VehicleDetails.Odometer, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.statusG()
	return res.VehicleLocation.Latitude, res.VehicleLocation.Longitude, err
}

// var _ api.Resurrector = (*Provider)(nil)

// // WakeUp implements the api.Resurrector interface
// func (v *Provider) WakeUp() error {
// 	return v.refreshG()
// }
