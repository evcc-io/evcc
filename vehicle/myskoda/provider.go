package myskoda

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Provider implements the vehicle api
type Provider struct {
	dataG  func() (VehicleResponse, error)
	action func(action string) error
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	return &Provider{
		dataG: util.Cached(func() (VehicleResponse, error) {
			return api.Vehicle(vin, "charging", "odometer", "airConditioning")
		}, cache),
		action: func(action string) error {
			return api.ChargeAction(vin, action)
		},
	}
}

// partError returns the error reported for the given response part
func partError(res VehicleResponse, part string) error {
	for _, e := range res.Errors {
		if strings.HasPrefix(e.Type, part) {
			// the api explains a permanent condition, don't log it on every update
			return fmt.Errorf("%s: %s: %w", e.Type, e.Description, api.ErrNotAvailable)
		}
	}
	return api.ErrNotAvailable
}

// charging returns the charging part of the vehicle data
func (v *Provider) charging() (*Charging, error) {
	res, err := v.dataG()
	if err != nil {
		return nil, err
	}
	if res.Vehicle.Charging == nil || res.Vehicle.Charging.Status == nil {
		return nil, partError(res, "CHARGING")
	}
	return res.Vehicle.Charging, nil
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.charging()
	if err != nil {
		return 0, err
	}
	return float64(res.Status.Battery.StateOfChargeInPercent), nil
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.charging()
	if err != nil {
		return api.StatusNone, err
	}

	switch s := res.Status.State; s {
	case "CONNECT_CABLE":
		return api.StatusA, nil
	case "READY_FOR_CHARGING", "CHARGING_INTERRUPTED", "DISCHARGING":
		return api.StatusB, nil
	// conserving is conservation charging
	case "CHARGING", "CONSERVING":
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %s", s)
	}
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.charging()
	if err != nil {
		return 0, err
	}
	return res.Status.Battery.RemainingCruisingRangeInMeters / 1e3, nil
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.charging()
	if err != nil {
		return time.Time{}, err
	}
	if !res.Status.FullyChargedAt.IsZero() {
		return res.Status.FullyChargedAt, nil
	}
	if res.Status.RemainingTimeToFullyChargedInMinutes > 0 {
		return time.Now().Add(time.Duration(res.Status.RemainingTimeToFullyChargedInMinutes) * time.Minute), nil
	}
	return time.Time{}, api.ErrNotAvailable
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.charging()
	if err != nil {
		return 0, err
	}
	if res.Settings == nil || res.Settings.TargetStateOfChargeInPercent == nil {
		return 0, api.ErrNotAvailable
	}
	return int64(*res.Settings.TargetStateOfChargeInPercent), nil
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	if res.Vehicle.Odometer == nil {
		return 0, partError(res, "ODOMETER")
	}
	return float64(res.Vehicle.Odometer.MileageInKm), nil
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res, err := v.dataG()
	if err != nil {
		return false, err
	}
	if res.Vehicle.AirConditioning == nil {
		return false, partError(res, "AIR_CONDITIONING")
	}
	return slices.Contains([]string{"COOLING", "HEATING", "HEATING_AUXILIARY", "VENTILATION"}, res.Vehicle.AirConditioning.State), nil
}

var _ api.ChargeController = (*Provider)(nil)

// ChargeEnable implements the api.ChargeController interface
func (v *Provider) ChargeEnable(enable bool) error {
	action := ActionStop
	if enable {
		action = ActionStart
	}
	return v.action(action)
}
