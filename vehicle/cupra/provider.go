package cupra

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

// Provider is an api.Vehicle implementation for Seat Cupra cars
type Provider struct {
	statusG func() (interface{}, error)
}

// NewProvider creates a new vehicle
func NewProvider(api *API, userID, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.NewCached(func() (interface{}, error) {
			return api.Status(userID, vin)
		}, cache).InterfaceGetter(),
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		return float64(res.Engines.Primary.Level), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		if res.Services.Charging.Status == "Connected" {
			status = api.StatusB
		}
		if res.Services.Charging.Status == "Charging" {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		rsc := res.Services.Charging
		if !rsc.Active {
			return time.Time{}, api.ErrNotAvailable
		}

		rt := rsc.RemainingTime
		if rsc.TargetPct > 0 && rsc.TargetPct < 100 {
			rt = rt * 100 / int64(rsc.TargetPct)
		}

		return time.Now().Add(time.Duration(rt) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		return int64(res.Engines.Primary.Range.Value), nil
	}

	return 0, err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		return res.Services.Climatisation.Active, 21, 21, nil
	}

	return active, 21, 21, err
}
