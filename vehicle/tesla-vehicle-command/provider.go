package vc

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

type Provider struct {
	dataG  func() (*VehicleData, error)
	wakeup func() (*VehicleData, error)
}

func NewProvider(api *API, vid int64, cache time.Duration) *Provider {
	impl := &Provider{
		dataG: provider.Cached(func() (*VehicleData, error) {
			res, err := api.VehicleData(vid)
			return res, apiError(err)
		}, cache),
		wakeup: func() (*VehicleData, error) {
			res, err := api.WakeUp(vid)
			return res, apiError(err)
		},
	}
	return impl
}

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	return float64(res.Response.ChargeState.UsableBatteryLevel), nil
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected
	res, err := v.dataG()
	if err != nil {
		return status, err
	}

	switch res.Response.ChargeState.ChargingState {
	case "Stopped", "NoPower", "Complete":
		status = api.StatusB
	case "Charging":
		status = api.StatusC
	}

	return status, nil
}

var _ api.ChargeRater = (*Provider)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (v *Provider) ChargedEnergy() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	return res.Response.ChargeState.ChargeEnergyAdded, nil
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	// miles to km
	return int64(kmPerMile * res.Response.ChargeState.BatteryRange), nil
}

var _ api.VehicleOdometer = (*Provider)(nil)

const kmPerMile = 1.609344

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	// miles to km
	return kmPerMile * res.Response.VehicleState.Odometer, nil
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.dataG()
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Add(time.Duration(res.Response.ChargeState.MinutesToFullCharge) * time.Minute), nil
}

// TODO api.Climater implementation has been removed as it drains battery. Re-check at a later time.

// var _ api.VehiclePosition = (*Provider)(nil)

// // Position implements the api.VehiclePosition interface
// func (v *Provider) Position() (float64, float64, error) {
// 	res, err := v.dataG()
// 	if err != nil {
// 		return 0, 0, err
// 	}
// 	if res.Response.DriveState.Latitude != 0 || res.Response.DriveState.Longitude != 0 {
// 		return res.Response.DriveState.Latitude, res.Response.DriveState.Longitude, nil
// 	}
// 	return res.Response.DriveState.ActiveRouteLatitude, res.Response.DriveState.ActiveRouteLongitude, nil
// }

var _ api.SocLimiter = (*Provider)(nil)

// TargetSoc implements the api.SocLimiter interface
func (v *Provider) TargetSoc() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	return float64(res.Response.ChargeState.ChargeLimitSoc), nil
}

var _ api.Resurrector = (*Provider)(nil)

// WakeUp implements the api.Resurrector interface
func (v *Provider) WakeUp() error {
	_, err := v.wakeup()
	return apiError(err)
}
