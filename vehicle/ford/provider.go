package ford

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/vehicle/ford/autonomic"
)

const refreshTimeout = time.Minute

type Provider struct {
	statusG     func() (autonomic.MetricsResponse, error)
	expiry      time.Duration
	refreshTime time.Time
	refreshId   string
	wakeup      func() error
}

func NewProvider(api *autonomic.API, vin string, expiry, cache time.Duration) *Provider {
	impl := &Provider{
		expiry: expiry,
	}

	impl.statusG = provider.Cached(func() (autonomic.MetricsResponse, error) {
		return api.Status(vin)
	}, cache)

	// impl.wakeup = func() error { return api.WakeUp(vin) }

	return impl
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	if err == nil {
		return res.Metrics.XevBatteryStateOfCharge.Value, nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err == nil {
		return int64(res.Metrics.XevBatteryRange.Value), nil
	}

	return 0, err
}

// var _ api.ChargeState = (*Provider)(nil)

// // Status implements the api.ChargeState interface
// func (v *Provider) Status() (api.ChargeStatus, error) {
// 	status := api.StatusA // disconnected

// 	res, err := v.statusG()
// 	if err == nil {
// 		if res.VehicleStatus.PlugStatus.Value == 1 {
// 			status = api.StatusB // connected, not charging
// 		}
// 		if res.VehicleStatus.ChargingStatus.Value == "ChargingAC" {
// 			status = api.StatusC // charging
// 		}
// 	}

// 	return status, err
// }

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	return res.Metrics.Odometer.Value, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.statusG()
	loc := res.Metrics.Position.Value.Location
	return loc.Lat, loc.Lon, err
}

// var _ api.Resurrector = (*Provider)(nil)

// // WakeUp implements the api.Resurrector interface
// func (v *Provider) WakeUp() error {
// 	return v.wakeup()
// }
