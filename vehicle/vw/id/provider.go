package id

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

// Provider is an api.Vehicle implementation for VW ID cars
type Provider struct {
	statusG func() (SelectiveSatus, error)
	action  func(action, value string) error
}

// NewProvider creates a new vehicle
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.Cached(func() (SelectiveSatus, error) {
			return api.Status(vin)
		}, cache),
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

	if err == nil && res.Charging != nil {
		return float64(res.Charging.BatteryStatus.Value.CurrentSOCPct), nil
	}

	return 0, fmt.Errorf("SoC not avaliable")
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if err == nil && res.Charging != nil {
		if res.Charging.PlugStatus.Value.PlugConnectionState == "connected" {
			status = api.StatusB
		}
		if res.Charging.ChargingStatus.Value.ChargingState == "charging" {
			status = api.StatusC
		}
		return status, nil
	}

	return "", fmt.Errorf("PlugStatus not avaliable")
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if err == nil && res.Charging != nil {
		return res.Charging.ChargingStatus.Value.CarCapturedTimestamp.Add(time.Duration(res.Charging.ChargingStatus.Value.RemainingChargingTimeToCompleteMin) * time.Minute), err
	}

	return time.Time{}, fmt.Errorf("FinishTime not avaliable")
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err == nil && res.Charging != nil {
		return int64(res.Charging.BatteryStatus.Value.CruisingRangeElectricKm), nil
	}

	return 0, fmt.Errorf("Range not avaliable")
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	return 0, fmt.Errorf("Odometer not avaliable")
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (active bool, outsideTemp, targetTemp float64, err error) {
	res, err := v.statusG()
	if err == nil && res.Climatisation != nil {
		state := strings.ToLower(res.Climatisation.ClimatisationStatus.Value.ClimatisationState)

		if state == "" {
			return false, 0, 0, api.ErrNotAvailable
		}

		active = state != "off" && state != "invalid" && state != "error"
		targetTemp = float64(res.Climatisation.ClimatisationSettings.Value.TargetTemperatureC)
		// TODO not available; use target temp to avoid wrong heating/cooling display
		outsideTemp = targetTemp

		return active, outsideTemp, targetTemp, nil
	}

	return false, 0, 0, fmt.Errorf("Climater not avaliable")
}

var _ api.SocLimiter = (*Provider)(nil)

// TargetSoC implements the api.SocLimiter interface
func (v *Provider) TargetSoC() (float64, error) {
	res, err := v.statusG()
	if err == nil && res.Charging != nil {
		return float64(res.Charging.ChargingSettings.Value.TargetSOCPct), nil
	}

	return 0, fmt.Errorf("Target SoC not avaliable")
}

var _ api.VehicleChargeController = (*Provider)(nil)

// StartCharge implements the api.VehicleChargeController interface
func (v *Provider) StartCharge() error {
	return v.action(ActionCharge, ActionChargeStart)
}

// StopCharge implements the api.VehicleChargeController interface
func (v *Provider) StopCharge() error {
	return v.action(ActionCharge, ActionChargeStop)
}
