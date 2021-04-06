package niu

import (
	"github.com/andig/evcc/api"
)

// API is a Niu API implementation
type API struct {
	apiG func() (interface{}, error)
}

// New returns a Niu API implementation
func New(apiG func() (interface{}, error)) *API {
	return &API{apiG: apiG}
}

var _ api.Battery = (*API)(nil)

// SoC implements the api.Vehicle interface
func (v *API) SoC() (float64, error) {
	res, err := v.apiG()

	if res, ok := res.(Response); err == nil && ok {
		return float64(res.Data.Batteries.CompartmentA.BatteryCharging), nil
	}

	return 0, err
}

// Status implements the Vehicle.Status interface
func (v *API) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.apiG()
	if res, ok := res.(Response); err == nil && ok {
		if res.Data.IsConnected {
			status = api.StatusB
		}
		if res.Data.IsCharging > 0 {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleRange = (*API)(nil)

// Range implements the api.VehicleRange interface
func (v *API) Range() (int64, error) {
	res, err := v.apiG()

	if res, ok := res.(Response); err == nil && ok {
		return int64(res.Data.EstimatedMileage), nil
	}

	return 0, err
}
