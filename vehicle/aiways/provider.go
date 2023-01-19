package aiways

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

// Provider implements the vehicle api
type Provider struct {
	statusG func() (StatusResponse, error)
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, user int64, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.Cached(func() (StatusResponse, error) {
			return api.Status(user, vin)
		}, cache),
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	if err == nil {
		return float64(res.Data.Vc.Soc), nil
	}
	return 0, err
}

// var _ api.ChargeState = (*Provider)(nil)

// // Status implements the api.ChargeState interface
// func (v *Provider) Status() (api.ChargeStatus, error) {
// 	status := api.StatusA // disconnected

// 	res, err := v.statusG()
// 	if err == nil {
// 		if res.Charger.Status.PlugStatusData.PlugState.Content == "connected" {
// 			status = api.StatusB
// 		}
// 		if res.Charger.Status.ChargingStatusData.ChargingState.Content == "charging" {
// 			status = api.StatusC
// 		}
// 	}

// 	return status, err
// }

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err == nil {
		return int64(res.Data.Vc.DrivingRange), nil
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	if err == nil {
		return res.Data.Vc.VehicleMileage, nil
	}

	return 0, err
}

// var _ api.VehiclePosition = (*Provider)(nil)

// // Position implements the api.VehiclePosition interface
// func (v *Provider) Position() (float64, float64, error) {
// 	res, err := v.statusG()
// 	if err == nil {
// 		return res.Data.Vc.Lat, res.Data.Vc.Lon, nil
// 	}

// 	return 0, 0, err
// }
