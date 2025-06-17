package polestar

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Provider struct {
	telemetryG func() (CarTelemetryData, error)
}

func NewProvider(log *util.Logger, api *API, vin string, timeout, cache time.Duration) *Provider {
	v := &Provider{
		telemetryG: util.Cached(func() (CarTelemetryData, error) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			return api.CarTelemetry(ctx, vin)
		}, cache),
	}

	return v
}

// SOC via car telemetry
func (v *Provider) Soc() (float64, error) {
	res, err := v.telemetryG()
	return res.Battery[0].BatteryChargeLevelPercentage, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status via car telemetry
func (v *Provider) Status() (api.ChargeStatus, error) {
	status, err := v.telemetryG()

	res := api.StatusA
	if status.Battery[0].ChargingStatus == "CHARGER_CONNECTION_STATUS_CONNECTED" {
		res = api.StatusB
	}
	if status.Battery[0].ChargingStatus == "CHARGING_STATUS_CHARGING" {
		res = api.StatusB
	}
	return res, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range via car telemetry
func (v *Provider) Range() (int64, error) {
	res, err := v.telemetryG()
	return res.Battery[0].EstimatedDistanceToEmptyKm, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer via car telemetry
func (v *Provider) Odometer() (float64, error) {
	res, err := v.telemetryG()
	return float64(res.Odometer[0].OdometerMeters) / 1e3, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime via car telemetry
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.telemetryG()
	return time.Now().Add(time.Duration(res.Battery[0].EstimatedChargingTimeToFullMinutes) * time.Minute), err
}
