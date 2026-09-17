package polestar

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Provider implements the vehicle api using the Polestar Data Portal
type Provider struct {
	batteryG   func() (Battery, error)
	odometerG  func() (Odometer, error)
	targetSocG func() (TargetSoc, error)
}

// NewProvider creates a Polestar Data Portal vehicle data provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	return &Provider{
		batteryG: util.Cached(func() (Battery, error) {
			return api.Battery(vin)
		}, cache),
		odometerG: util.Cached(func() (Odometer, error) {
			return api.Odometer(vin)
		}, cache),
		targetSocG: util.Cached(func() (TargetSoc, error) {
			return api.TargetSoc(vin)
		}, cache),
	}
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.batteryG()
	if err != nil {
		return 0, err
	}
	return res.BatteryChargeLevelPercentage, nil
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.batteryG()
	if err != nil {
		return api.StatusNone, err
	}

	switch res.ChargingStatusV2 {
	case "CHARGING_STATUS_CHARGING", "CHARGING_STATUS_SMART_CHARGING":
		return api.StatusC, nil
	}

	if res.ChargerConnectionStatus == "CHARGER_CONNECTION_STATUS_CONNECTED" {
		return api.StatusB, nil
	}

	return api.StatusA, nil
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.batteryG()
	if err != nil {
		return 0, err
	}
	return res.EstimatedDistanceToEmptyKm, nil
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.odometerG()
	if err != nil {
		return 0, err
	}
	return res.OdometerMeters / 1e3, nil
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.batteryG()
	if err != nil {
		return time.Time{}, err
	}

	if res.EstimatedChargingTimeToFullMinutes <= 0 {
		return time.Time{}, api.ErrNotAvailable
	}

	// anchor the relative remaining time to the API's capture timestamp
	base := res.Timestamp.Time()
	if base.IsZero() {
		base = time.Now()
	}

	return base.Add(time.Duration(res.EstimatedChargingTimeToFullMinutes) * time.Minute), nil
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.targetSocG()
	if err != nil {
		return 0, err
	}
	return res.TargetSoc.BatteryChargeTargetLevel, nil
}
