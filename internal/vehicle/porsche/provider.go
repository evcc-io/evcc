package porsche

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// Provider is an api.Vehicle implementation for PSA cars
type Provider struct {
	emobilityVehicle bool
	statusG          func() (interface{}, error)
}

// NewProvider creates a new vehicle
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		emobilityVehicle: api.emobilityVehicle,
		statusG: provider.NewCached(func() (interface{}, error) {
			return api.Status(vin)
		}, cache).InterfaceGetter(),
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.statusG()
	if v.emobilityVehicle {
		if res, ok := res.(porscheEmobilityResponse); err == nil && ok {
			return float64(res.BatteryChargeStatus.StateOfChargeInPercentage), nil
		}
	} else {
		if res, ok := res.(porscheVehicleResponse); err == nil && ok {
			return res.CarControlData.BatteryLevel.Value, nil
		}
	}

	return 0, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if v.emobilityVehicle {
		if res, ok := res.(porscheEmobilityResponse); err == nil && ok {
			return int64(res.BatteryChargeStatus.RemainingERange.ValueInKilometers), nil
		}
	} else {
		if res, ok := res.(porscheVehicleResponse); err == nil && ok {
			return int64(res.CarControlData.RemainingRanges.ElectricalRange.Distance.Value), nil
		}
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if v.emobilityVehicle {
		if res, ok := res.(*porscheEmobilityResponse); err == nil && ok {
			t := time.Now()
			return t.Add(time.Duration(res.BatteryChargeStatus.RemainingChargeTimeUntil100PercentInMinutes) * time.Minute), err
		}
	}

	return time.Time{}, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.statusG()
	if v.emobilityVehicle {
		if res, ok := res.(porscheEmobilityResponse); err == nil && ok {
			switch res.BatteryChargeStatus.PlugState {
			case "DISCONNECTED":
				return api.StatusA, nil
			case "CONNECTED":
				switch res.BatteryChargeStatus.ChargingState {
				case "OFF", "COMPLETED":
					return api.StatusB, nil
				case "ON":
					return api.StatusC, nil
				}
			}
		}
	} else {
		return api.StatusNone, err
	}

	return api.StatusNone, err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	res, err := v.statusG()
	if v.emobilityVehicle {
		if res, ok := res.(porscheEmobilityResponse); err == nil && ok {
			switch res.DirectClimatisation.ClimatisationState {
			case "OFF":
				return false, 0, 0, nil
			case "ON":
				return true, 0, 0, nil
			}
		}
	}

	return active, outsideTemp, targetTemp, err
}
