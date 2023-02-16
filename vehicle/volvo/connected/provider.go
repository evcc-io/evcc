package connected

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

// Provider implements the vehicle api
type Provider struct {
	statusG func() (RechargeStatus, error)
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.Cached(func() (RechargeStatus, error) {
			return api.RechargeStatus(vin)
		}, cache),
	}
	return impl
}

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	return res.Data.BatteryChargeLevel.Value, err
}

// Range implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if err != nil {
		return status, nil
	}

	switch res.Data.ChargingConnectionStatus.Value {
	case "CONNECTION_STATUS_DISCONNECTED":
		status = api.StatusA
	case "CONNECTION_STATUS_CONNECTED_AC", "CONNECTION_STATUS_CONNECTED_DC":
		status = api.StatusB
	}

	if res.Data.ChargingSystemStatus.Value == "CHARGING_SYSTEM_CHARGING" {
		status = api.StatusC
	}

	return status, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (rng int64, err error) {
	res, err := v.statusG()
	return res.Data.ElectricRange.Value, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	return res.Data.EstimatedChargingTime.Timestamp.Add(time.Duration(res.Data.EstimatedChargingTime.Value) * time.Minute), err
}
