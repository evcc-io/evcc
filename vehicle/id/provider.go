package id

import (
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// Provider is an api.Vehicle implementation for VW ID cars
type Provider struct {
	statusG           func() (interface{}, error)
	startChargeAction func() error
}

// NewProvider creates a new vehicle
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.NewCached(func() (interface{}, error) {
			return api.Status(vin)
		}, cache).InterfaceGetter(),
		startChargeAction: func() error {
			return api.Action(vin, ActionCharge, ActionChargeStart)
		},
	}
	return impl
}

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		return float64(res.Data.BatteryStatus.CurrentSOCPercent), nil
	}

	return 0, err
}

// Status implements the Vehicle.Status interface
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

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		return int64(res.Data.BatteryStatus.CruisingRangeElectricKm), nil
	}

	return 0, err
}

// Climater implements the Vehicle.Climater interface
func (v *Provider) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	res, err := v.statusG()
	if res, ok := res.(Status); err == nil && ok {
		state := strings.ToLower(res.Data.ClimatisationStatus.ClimatisationState)

		if state == "" {
			return false, 0, 0, api.ErrNotAvailable
		}

		active := state != "off" && state != "invalid" && state != "error"

		targetTemp = res.Data.ClimatisationSettings.TargetTemperatureC

		// TODO: not available; use target temp to avoid wrong heating/cooling display
		outsideTemp = targetTemp

		return active, outsideTemp, targetTemp, nil
	}

	return active, outsideTemp, targetTemp, err
}

// StartCharge implements the VehicleStartCharge interface
func (v *Provider) StartCharge() error {
	return v.startChargeAction()
}
