package hello

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

// https://github.com/TA2k/ioBroker.smart-eq

type Provider struct {
	statusG func() (VehicleStatus, error)
}

func NewProvider(log *util.Logger, api *API, vin string, cache time.Duration) *Provider {
	v := &Provider{
		statusG: provider.Cached(func() (VehicleStatus, error) {
			return api.Status(vin)
		}, cache),
	}

	return v
}

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	return float64(res.AdditionalVehicleStatus.ElectricVehicleStatus.ChargeLevel), err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	return int64(res.AdditionalVehicleStatus.ElectricVehicleStatus.DistanceToEmptyOnBatteryOnly), err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.statusG()
	const div = 3600000.0
	return float64(res.BasicVehicleStatus.Position.Latitude) / div, float64(res.BasicVehicleStatus.Position.Longitude) / div, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the Provider.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	return res.AdditionalVehicleStatus.MaintenanceStatus.Odometer, err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res, err := v.statusG()
	return res.AdditionalVehicleStatus.ClimateStatus.PreClimateActive || res.AdditionalVehicleStatus.ClimateStatus.Defrost, err
}
