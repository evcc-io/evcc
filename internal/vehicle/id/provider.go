package id

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// Provider is an api.Vehicle implementation for VW ID cars
type Provider struct {
	statusG func() (interface{}, error)
	action  func(action, value string) error
}

// NewProvider creates a new vehicle
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.NewCached(func() (interface{}, error) {
			return api.Status(vin)
		}, cache).InterfaceGetter(),
		action: func(action, value string) error {
			return api.Action(vin, action, value)
		},
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		return float64(res.Data.BatteryStatus.CurrentSOCPercent), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		if res.Data.PlugStatus.PlugConnectionState == "connected" {
			status = api.StatusB
		}
		if res.Data.ChargingStatus.ChargingState == "charging" {
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
		cst := res.Data.ChargingStatus
		timestamp, err := time.Parse(time.RFC3339, cst.CarCapturedTimestamp)
		return timestamp.Add(time.Duration(cst.RemainingChargingTimeToCompleteMin) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		return int64(res.Data.BatteryStatus.CruisingRangeElectricKm), nil
	}

	return 0, err
}

var _ api.VehicleStartCharge = (*Provider)(nil)

// StartCharge implements the api.VehicleStartCharge interface
func (v *Provider) StartCharge() error {
	return v.action(ActionCharge, ActionChargeStart)
}

var _ api.VehicleStopCharge = (*Provider)(nil)

// StopCharge implements the api.VehicleStopCharge interface
func (v *Provider) StopCharge() error {
	return v.action(ActionCharge, ActionChargeStop)
}
