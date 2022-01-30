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
	statusG    func() (interface{}, error)
	emobilityG func() (interface{}, error)
	mobileG    func() (interface{}, error)
}

// NewProvider creates a new vehicle
func NewProvider(log *util.Logger, api *API, emobility *EmobilityAPI, mobile *MobileAPI, vin string, capabilities CapabilitiesResponse, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.NewCached(func() (interface{}, error) {
			return api.Status(vin)
		}, cache).InterfaceGetter(),

		emobilityG: provider.NewCached(func() (interface{}, error) {
			if capabilities.CarModel != "" {
				return emobility.Status(vin, capabilities.CarModel)
			} else {
				return EmobilityResponse{}, errors.New("no car model")
			}
		}, cache).InterfaceGetter(),

		mobileG: provider.NewCached(func() (interface{}, error) {
			return mobile.Status(vin)
		}, cache).InterfaceGetter(),
	}

	return impl
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.mobileG()
	if res, ok := res.(StatusResponseMobile); err == nil && ok {
		for _, m := range res.Measurements {
			if m.Key == "BATTERY_LEVEL" {
				if !m.Status.IsEnabled {
					return 0, errors.New(m.Status.Cause)
				}
				return float64(m.Value.Percent), nil
			}
		}
	}

	res, err = v.emobilityG()
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
	res, err := v.mobileG()
	if res, ok := res.(StatusResponseMobile); err == nil && ok {
		for _, m := range res.Measurements {
			if m.Key == "E_RANGE" {
				if !m.Status.IsEnabled {
					return 0, errors.New(m.Status.Cause)
				}
				return int64(m.Value.Kilometers), nil
			}
		}
	}

	res, err = v.emobilityG()
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
	res, err := v.mobileG()
	if res, ok := res.(StatusResponseMobile); err == nil && ok {
		for _, m := range res.Measurements {
			if m.Key == "BATTERY_CHARGING_STATE" {
				if !m.Status.IsEnabled {
					return time.Time{}, errors.New(m.Status.Cause)
				}
				if m.Value.EndsAt == "" {
					if m.Value.LastModified != "" {
						return time.Parse(time.RFC3339, m.Value.LastModified)
					}
					return time.Time{}, nil
				}
				return time.Parse(time.RFC3339, m.Value.EndsAt)
			}
		}
	}

	res, err = v.emobilityG()
	if res, ok := res.(EmobilityResponse); err == nil && ok {
		return time.Now().Add(time.Duration(res.BatteryChargeStatus.RemainingChargeTimeUntil100PercentInMinutes) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.mobileG()
	if res, ok := res.(StatusResponseMobile); err == nil && ok {
		for _, m := range res.Measurements {
			if m.Key == "BATTERY_CHARGING_STATE" {
				if !m.Status.IsEnabled {
					return api.StatusNone, errors.New(m.Status.Cause)
				}
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
	}

	res, err = v.emobilityG()
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
			case "ON", "CHARGING":
				return api.StatusC, nil
			default:
				return api.StatusNone, errors.New("emobility - unknown charging state: " + res.BatteryChargeStatus.ChargingState)
			}
		}
	}

	return api.StatusNone, err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	res, err := v.mobileG()
	if res, ok := res.(StatusResponseMobile); err == nil && ok {
		for _, m := range res.Measurements {
			if m.Key == "CLIMATIZER_STATE" {
				if !m.Status.IsEnabled {
					return active, 20, 20, errors.New(m.Status.Cause)
				}
				return m.Value.IsOn, 20, 20, err
			}
		}
	}

	res, err = v.emobilityG()
	if res, ok := res.(EmobilityResponse); err == nil && ok {
		switch res.DirectClimatisation.ClimatisationState {
		case "OFF":
			return false, 20, 20, nil
		case "ON":
			return true, 20, 20, nil
		default:
			return active, outsideTemp, targetTemp, errors.New("emobility - unknown climate state: " + res.DirectClimatisation.ClimatisationState)
		}
	}

	return active, outsideTemp, targetTemp, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.mobileG()
	if res, ok := res.(StatusResponseMobile); err == nil && ok {
		for _, m := range res.Measurements {
			if m.Key == "MILEAGE" {
				if !m.Status.IsEnabled {
					return 0, errors.New(m.Status.Cause)
				}
				return float64(m.Value.Kilometers), nil
			}
		}
	}

	res, err = v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		return res.Mileage.Value, nil
	}

	return 0, err
}
