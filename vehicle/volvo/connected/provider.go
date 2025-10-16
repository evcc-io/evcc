package connected

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

// Provider implements the vehicle api
type Provider struct {
	statusG func() (EnergyState, error)
	odoG    func() (OdometerState, error)
}

func tokenGuard[T any](fun func(string) (T, error), ts oauth2.TokenSource, vin string) (T, error) {
	// don't try as long as there's no token
	if _, err := ts.Token(); err != nil {
		var zero T
		return zero, api.ErrNotAvailable
	}
	return fun(vin)
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, ts oauth2.TokenSource, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: util.Cached(func() (EnergyState, error) {
			return tokenGuard(api.EnergyState, ts, vin)
		}, cache),
		odoG: util.Cached(func() (OdometerState, error) {
			return tokenGuard(api.OdometerState, ts, vin)
		}, cache),
	}
	return impl
}

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	return res.BatteryChargeLevel.Value, err
}

// Range implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.statusG()
	if err != nil {
		return api.StatusNone, err
	}

	status := api.StatusA // disconnected

	switch s := res.ChargerConnectionStatus.Value; s {
	case "CONNECTED":
		status = api.StatusB
	case "FAULT":
		return status, fmt.Errorf("invalid status: %s", s)
	}

	if res.ChargingStatus.Value == "CHARGING" {
		status = api.StatusC
	}

	return status, nil
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (rng int64, err error) {
	res, err := v.statusG()
	return res.ElectricRange.Value, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	return res.EstimatedChargingTimeTimeToTargetBatteryChargeLevel.Timestamp.Add(time.Duration(res.EstimatedChargingTimeTimeToTargetBatteryChargeLevel.Value) * time.Minute), err
}

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.statusG()

	return int64(res.TargetBatteryChargeLevel.Value), err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.odoG()
	return float64(res.Data.Odometer.Value), err
}
