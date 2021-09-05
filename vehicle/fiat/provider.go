package fiat

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

type Provider struct {
	// api     *API
	statusG func() (interface{}, error)
}

func NewProvider(api *API, vin, pin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.NewCached(func() (interface{}, error) {
			return api.Status(vin)
		}, cache).InterfaceGetter(),
	}

	if pin != "" {
		res, err := api.Status(vin)
		if err != nil {
			panic(err)
		}

		fmt.Println(res.Timestamp)
		fmt.Println(res.EvInfo.Timestamp)
		fmt.Println(res.VehicleInfo.Timestamp)

		api.Action(vin, pin, "ev", "DEEPREFRESH")
		panic(1)
	}

	return impl
}

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		return float64(res.EvInfo.Battery.StateOfCharge), nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		return int64(res.EvInfo.Battery.DistanceToEmpty.Value), nil
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		return float64(res.VehicleInfo.Odometer.Odometer.Value), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		if res.EvInfo.Battery.PlugInStatus {
			status = api.StatusB // connected, not charging
		}
		if res.EvInfo.Battery.ChargingStatus == "CHARGING" {
			status = api.StatusC // charging
		}
	}

	return status, err
}
