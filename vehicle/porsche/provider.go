package porsche

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

// Provider is an api.Vehicle implementation for Porsche PHEV cars
type Provider struct {
	statusG    func() (interface{}, error)
	emobilityG func() (interface{}, error)
}

// NewProvider creates a new vehicle
func NewProvider(log *util.Logger, api *API, emobility *EmobilityAPI, vin, model string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.NewCached(func() (interface{}, error) {
			return api.Status(vin)
		}, cache).InterfaceGetter(),

		emobilityG: provider.NewCached(func() (interface{}, error) {
			return emobility.Status(vin, model)
		}, cache).InterfaceGetter(),
	}

	return impl
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.emobilityG()
	if res, ok := res.(EmobilityResponse); err == nil && ok {
		return float64(res.BatteryChargeStatus.StateOfChargeInPercentage), nil
	}

	res, err = v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		return res.BatteryLevel.Value, nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.emobilityG()
	if res, ok := res.(EmobilityResponse); err == nil && ok {
		return res.BatteryChargeStatus.RemainingERange.ValueInKilometers, nil
	}

	res, err = v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		return int64(res.RemainingRanges.ElectricalRange.Distance.Value), nil
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.emobilityG()
	if res, ok := res.(EmobilityResponse); err == nil && ok {
		return time.Now().Add(time.Duration(res.BatteryChargeStatus.RemainingChargeTimeUntil100PercentInMinutes) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.emobilityG()
	if res, ok := res.(EmobilityResponse); err == nil && ok {
		switch res.BatteryChargeStatus.PlugState {
		case "DISCONNECTED":
			return api.StatusA, nil
		case "CONNECTED":
			// ignore if the car is connected to a DC charging station
			if res.BatteryChargeStatus.ChargingInDCMode {
				return api.StatusA, nil
			}
			switch res.BatteryChargeStatus.ChargingState {
			case "ERROR":
				return api.StatusF, nil
			case "OFF", "COMPLETED":
				return api.StatusB, nil
			case "ON":
				return api.StatusC, nil
			}
		}
	}

	return api.StatusNone, err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	res, err := v.emobilityG()
	if res, ok := res.(EmobilityResponse); err == nil && ok {
		switch res.DirectClimatisation.ClimatisationState {
		case "OFF":
			return false, 20, 20, nil
		case "ON":
			return true, 20, 20, nil
		}
	}

	return active, outsideTemp, targetTemp, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		return res.Mileage.Value, nil
	}

	return 0, err
}
