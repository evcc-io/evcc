package polestar

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

// https://github.com/TA2k/ioBroker.polestar

type Provider struct {
	statusG func() (BatteryData, error)
	odoG    func() (OdometerData, error)
}

func NewProvider(log *util.Logger, api *API, vin string, timeout, cache time.Duration) *Provider {
	v := &Provider{
		statusG: provider.Cached(func() (BatteryData, error) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			return api.Status(ctx, vin)
		}, cache),
		odoG: provider.Cached(func() (OdometerData, error) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			return api.Odometer(ctx, vin)
		}, cache),
	}

	return v
}

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	return float64(res.BatteryChargeLevelPercentage), err
}

var _ api.ChargeState = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status, err := v.statusG()

	res := api.StatusA
	if status.ChargerConnectionStatus == "CHARGER_CONNECTION_STATUS_CONNECTED" {
		res = api.StatusB
	}
	if status.ChargingStatus == "CHARGING_STATUS_CHARGING" {
		res = api.StatusB
	}

	return res, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	return int64(res.EstimatedDistanceToEmptyKm), err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the Provider.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.odoG()
	return float64(res.OdometerMeters) / 1e3, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Add(time.Duration(res.EstimatedChargingTimeToFullMinutes) * time.Minute), nil
}
