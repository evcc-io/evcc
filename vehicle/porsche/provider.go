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
	mobileG    func() (StatusResponseMobile, error)
}

// NewProvider creates a vehicle api provider
func NewProvider(log *util.Logger, api *API, emobility *EmobilityAPI, mobile *MobileAPI, vin, carModel string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.Cached(func() (StatusResponse, error) {
			return api.Status(vin)
		}, cache),

		emobilityG: provider.Cached(func() (EmobilityResponse, error) {
			if carModel != "" {
				return emobility.Status(vin, carModel)
			}
			return EmobilityResponse{}, errors.New("no car model")
		}, cache),

		mobileG: provider.Cached(func() (StatusResponseMobile, error) {
			return mobile.Status(vin, []string{BATTERY_LEVEL, BATTERY_CHARGING_STATE, CLIMATIZER_STATE, E_RANGE, HEATING_STATE, MILEAGE})
		}, cache),
	}

	return impl
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.mobileG()
	if err == nil {
		m, err := res.MeasurementByKey("BATTERY_LEVEL")
		if err != nil && err != api.ErrNotAvailable {
			return 0, err
		}
		if err != api.ErrNotAvailable {
			return float64(m.Value.Percent), nil
		}
	}

	res3, err := v.emobilityG()
	if err == nil {
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
	res, err := v.mobileG()
	if err == nil {
		m, err := res.MeasurementByKey("E_RANGE")
		if err != nil && err != api.ErrNotAvailable {
			return 0, err
		}
		if err != api.ErrNotAvailable {
			return int64(m.Value.Kilometers), nil
		}
	}

	res3, err := v.emobilityG()
	if err == nil {
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
	res, err := v.mobileG()
	if err == nil {
		m, err := res.MeasurementByKey("BATTERY_CHARGING_STATE")
		if err != nil && err != api.ErrNotAvailable {
			return time.Time{}, err
		}

		if err != api.ErrNotAvailable {
			if m.Value.EndsAt == "" {
				if m.Value.LastModified != "" {
					return time.Parse(time.RFC3339, m.Value.LastModified)
				}
				return time.Time{}, nil
			}
			return time.Parse(time.RFC3339, m.Value.EndsAt)
		}
	}

	res2, err := v.emobilityG()
	if err == nil {
		return time.Now().Add(time.Duration(res2.BatteryChargeStatus.RemainingChargeTimeUntil100PercentInMinutes) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.mobileG()
	if err == nil {
		m, err := res.MeasurementByKey("BATTERY_CHARGING_STATE")
		if err != nil && err != api.ErrNotAvailable {
			return api.StatusNone, err
		}

		if err != api.ErrNotAvailable {
			switch m.Value.Status {
			case "FAST_CHARGING", "NOT_PLUGGED", "UNKNOWN":
				return api.StatusA, nil
			case "CHARGING_COMPLETED", "CHARGING_PAUSED", "READY_TO_CHARGE", "SOC_REACHED",
				"INITIALISING", "STANDBY", "SUSPENDED", "PLUGGED_LOCKED", "PLUGGED_NOT_LOCKED":
				return api.StatusB, nil
			case "CHARGING":
				return api.StatusC, nil
			case "CHARGING_ERROR":
				return api.StatusF, nil
			default:
				return api.StatusNone, errors.New("mobile - unknown charging status: " + m.Value.Status)
			}
		}
	}

	res2, err := v.emobilityG()
	if err == nil {
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
	res, err := v.mobileG()
	if err == nil {
		m, err := res.MeasurementByKey("CLIMATIZER_STATE")
		if err != nil && err != api.ErrNotAvailable {
			return false, err
		}
		if err != api.ErrNotAvailable {
			return m.Value.IsOn, err
		}
	}

	res2, err := v.emobilityG()
	if err == nil {
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
	res, err := v.mobileG()
	if err == nil {
		m, err := res.MeasurementByKey("MILEAGE")
		if err != nil && err != api.ErrNotAvailable {
			return 0, err
		}
		if err != api.ErrNotAvailable {
			return float64(m.Value.Kilometers), nil
		}
	}

	res2, err := v.statusG()
	if err == nil {
		return res2.Mileage.Value, nil
	}

	return 0, err
}
