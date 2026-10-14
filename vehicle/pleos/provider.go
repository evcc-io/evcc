package pleos

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Provider implements the vehicle api using the Pleos Vehicle Data API
type Provider struct {
	batteryG    func() (Battery, error)
	powertrainG func() (Powertrain, error)
	drivingG    func() (Driving, error)
	locationG   func() (Location, error)
	statusG     func() (Status, error)
}

// NewProvider creates a Pleos vehicle data provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	return &Provider{
		batteryG: util.Cached(func() (Battery, error) {
			return api.Battery(vin)
		}, cache),
		powertrainG: util.Cached(func() (Powertrain, error) {
			return api.Powertrain(vin)
		}, cache),
		drivingG: util.Cached(func() (Driving, error) {
			return api.Driving(vin)
		}, cache),
		locationG: util.Cached(func() (Location, error) {
			return api.Location(vin)
		}, cache),
		statusG: util.Cached(func() (Status, error) {
			return api.Status(vin)
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
	return res.Charge.StateOfCharge, nil
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.batteryG()
	if err != nil {
		return api.StatusNone, err
	}

	if res.Charge.Charging {
		return api.StatusC, nil
	}

	switch res.Charge.Status {
	case "charging", "fastCharging", "wirelessCharging":
		return api.StatusC, nil
	}

	if res.Charge.Plugin == "connected" {
		return api.StatusB, nil
	}

	return api.StatusA, nil
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.powertrainG()
	if err != nil {
		return 0, err
	}
	return res.Range()
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.drivingG()
	if err != nil {
		return 0, err
	}
	return res.Odometer.Km()
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.batteryG()
	if err != nil {
		return 0, err
	}
	if res.Charge.TargetStateOfCharge.Standard == 0 {
		return 0, api.ErrNotAvailable
	}
	return res.Charge.TargetStateOfCharge.Standard, nil
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.locationG()
	if err != nil {
		return 0, 0, err
	}
	return res.Location.Latitude, res.Location.Longitude, nil
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res, err := v.statusG()
	if err != nil {
		return false, err
	}
	return res.ClimateControl.Status == "on", nil
}
