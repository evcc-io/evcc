package id

import (
	"time"

	"github.com/andig/evcc/provider"
)

// Provider is an api.Vehicle implementation for VW ID cars
type Provider struct {
	statusG func() (interface{}, error)
}

// NewProvider creates a new vehicle
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.NewCached(func() (interface{}, error) {
			return api.Status(vin)
		}, cache).InterfaceGetter(),
	}
	return impl
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Provider) ChargeState() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		return float64(res.Data.BatteryStatus.CurrentSOCPercent), nil
	}

	return 0, err
}

// FinishTime implements the Vehicle.ChargeFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		cst := res.Data.ChargingStatus
		timestamp, err := time.Parse(time.RFC3339, cst.CarCapturedTimestamp)
		return timestamp.Add(time.Duration(cst.RemainingChargingTimeToCompleteMin) * time.Minute), err
	}

	return time.Time{}, err
}

// Range implements the Vehicle.Range interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		return int64(res.Data.BatteryStatus.CruisingRangeElectricKm), nil
	}

	return 0, err
}
