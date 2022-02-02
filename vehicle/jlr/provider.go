package jlr

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

type Provider struct {
	statusG func() (interface{}, error)
}

func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.NewCached(func() (interface{}, error) {
			return api.Status(vin)
		}, cache).InterfaceGetter(),
	}

	return impl
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Battery interface
func (v *Provider) SoC() (float64, error) {
	var val float64
	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		val, err = res.VehicleStatus.EvStatus.FloatVal("EV_RANGE_VSC_INITIAL_HV_BATT_ENERGYx100")
	}

	return val, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	var val int64
	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		val, err = res.VehicleStatus.EvStatus.IntVal("EV_RANGE_ON_BATTERY_KM")
	}

	return val, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		if s, err := res.VehicleStatus.EvStatus.StringVal("EV_IS_PLUGGED_IN"); err == nil && s == "CONNECTED" {
			// fmt.Println("EV_IS_PLUGGED_IN", s, err)
			status = api.StatusB
		}

		if s, err := res.VehicleStatus.EvStatus.StringVal("EV_CHARGING_STATUS"); err == nil && s == "CHARGING" {
			// fmt.Println("EV_CHARGING_STATUS", s, err)
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	var val float64
	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		val, err = res.VehicleStatus.CoreStatus.FloatVal("ODOMETER")
	}

	return val / 1e3, err
}
