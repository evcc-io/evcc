package connected

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Provider implements the vehicle api
type Provider struct {
	statusG func() (EnergyState, error)
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: util.Cached(func() (EnergyState, error) {
			return api.EnergyState(vin)
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
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if err != nil {
		return status, nil
	}

	switch res.ChargingConnectionStatus.Value {
	case "DISCONNECTED":
		status = api.StatusA
	case "CONNECTED", "FAULT":
		status = api.StatusB
	}

	if res.ChargingStatus.Status == "CHARGING" {
		status = api.StatusC
	}

	return status, err
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
