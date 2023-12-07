package porsche

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

// Provider is an api.Vehicle implementation for Porsche PHEV cars
type Provider struct {
	statusG    func() (StatusResponse, error)
	emobilityG func() (EmobilityResponse, error)
	wakeup     func() error
}

// NewProvider creates a vehicle api provider
func NewProvider(log *util.Logger, connect *API, emobility *EmobilityAPI, vin, carModel string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.Cached(func() (StatusResponse, error) {
			return connect.Status(vin)
		}, cache),

		emobilityG: provider.Cached(func() (EmobilityResponse, error) {
			if carModel != "" {
				return emobility.Status(vin, carModel)
			}
			return EmobilityResponse{}, api.ErrNotAvailable
		}, cache),

		wakeup: func() error {
			return connect.WakeUp(vin)
		},
	}

	return impl
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res3, err := v.emobilityG()
	if err == nil && res3.BatteryChargeStatus != nil {
		return float64(res3.BatteryChargeStatus.StateOfChargeInPercentage), nil
	}

	res2, err := v.statusG()
	if err == nil {
		return res2.BatteryLevel.Value, nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res3, err := v.emobilityG()
	if err == nil && res3.BatteryChargeStatus != nil {
		return res3.BatteryChargeStatus.RemainingERange.ValueInKilometers, nil
	}

	res2, err := v.statusG()
	if err == nil {
		return int64(res2.RemainingRanges.ElectricalRange.Distance.Value), nil
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res2, err := v.emobilityG()
	if err == nil {
		if res2.BatteryChargeStatus == nil {
			return time.Time{}, api.ErrNotAvailable
		}

		return time.Now().Add(time.Duration(res2.BatteryChargeStatus.RemainingChargeTimeUntil100PercentInMinutes) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res2, err := v.emobilityG()
	if err == nil {
		if res2.BatteryChargeStatus == nil {
			return api.StatusNone, api.ErrNotAvailable
		}

		switch res2.BatteryChargeStatus.PlugState {
		case "DISCONNECTED":
			return api.StatusA, nil
		case "CONNECTED":
			// ignore if the car is connected to a DC charging station
			if res2.BatteryChargeStatus.ChargingInDCMode {
				return api.StatusA, nil
			}
			switch res2.BatteryChargeStatus.ChargingState {
			case "ERROR":
				return api.StatusF, nil
			case "OFF", "COMPLETED":
				return api.StatusB, nil
			case "ON", "CHARGING":
				return api.StatusC, nil
			default:
				return api.StatusNone, errors.New("emobility - unknown charging state: " + res2.BatteryChargeStatus.ChargingState)
			}
		}
	}

	return api.StatusNone, err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res2, err := v.emobilityG()
	if err == nil {
		if res2.BatteryChargeStatus == nil {
			return false, api.ErrNotAvailable
		}

		switch res2.DirectClimatisation.ClimatisationState {
		case "OFF":
			return false, nil
		case "ON":
			return true, nil
		default:
			return false, errors.New("emobility - unknown climate state: " + res2.DirectClimatisation.ClimatisationState)
		}
	}

	return false, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res2, err := v.statusG()
	if err == nil {
		return res2.Mileage.Value, nil
	}

	return 0, err
}

var _ api.Resurrector = (*Provider)(nil)

// WakeUp implements the api.Resurrector interface
func (v *Provider) WakeUp() error {
	return v.wakeup()
}
