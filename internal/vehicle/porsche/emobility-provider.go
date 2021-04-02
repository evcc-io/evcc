package porsche

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

type porscheEmobilityResponse struct {
	BatteryChargeStatus struct {
		ChargeRate struct {
			Unit             string
			Value            float64
			ValueInKmPerHour int64
		}
		ChargingInDCMode                            bool
		ChargingMode                                string
		ChargingPower                               float64
		ChargingReason                              string
		ChargingState                               string
		ChargingTargetDateTime                      string
		ExternalPowerSupplyState                    string
		PlugState                                   string
		RemainingChargeTimeUntil100PercentInMinutes int64
		StateOfChargeInPercentage                   int64
		RemainingERange                             struct {
			OriginalUnit      string
			OriginalValue     int64
			Unit              string
			Value             int64
			ValueInKilometers int64
		}
	}
	ChargingStatus string
	DirectCharge   struct {
		Disabled bool
		IsActive bool
	}
	DirectClimatisation struct {
		ClimatisationState         string
		RemainingClimatisationTime int64
	}
}

// EMobility Provider is an api.Vehicle implementation for PSA cars
type EMobilityProvider struct {
	api     *API
	statusG func() (interface{}, error)
}

// NewEMobilityProvider creates a new vehicle
func NewEMobilityProvider(api *API, vin string, cache time.Duration) *EMobilityProvider {
	impl := &EMobilityProvider{
		api: api,
	}

	impl.statusG = provider.NewCached(func() (interface{}, error) {
		return impl.status(vin)
	}, cache).InterfaceGetter()

	return impl
}

// Status implements the vehicle status repsonse
func (v *EMobilityProvider) status(vin string) (interface{}, error) {
	uri := fmt.Sprintf("https://api.porsche.com/service-vehicle/de/de_DE/e-mobility/J1/%s?timezone=Europe/Berlin", vin)
	req, err := v.api.request(uri, true)
	if err != nil {
		return 0, err
	}

	req.Header.Set("apikey", v.api.emobilityClientID)
	var pr porscheEmobilityResponse
	err = v.api.DoJSON(req, &pr)

	return pr, err
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *EMobilityProvider) SoC() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(porscheEmobilityResponse); err == nil && ok {
		return float64(res.BatteryChargeStatus.StateOfChargeInPercentage), nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *EMobilityProvider) Range() (int64, error) {
	res, err := v.statusG()
	if res, ok := res.(porscheEmobilityResponse); err == nil && ok {
		return int64(res.BatteryChargeStatus.RemainingERange.ValueInKilometers), nil
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*EMobilityProvider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *EMobilityProvider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if res, ok := res.(*porscheEmobilityResponse); err == nil && ok {
		t := time.Now()
		return t.Add(time.Duration(res.BatteryChargeStatus.RemainingChargeTimeUntil100PercentInMinutes) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.ChargeState = (*EMobilityProvider)(nil)

// Status implements the api.ChargeState interface
func (v *EMobilityProvider) Status() (api.ChargeStatus, error) {
	res, err := v.statusG()
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

	return api.StatusNone, err
}

var _ api.VehicleClimater = (*EMobilityProvider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *EMobilityProvider) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	res, err := v.statusG()
	if res, ok := res.(porscheEmobilityResponse); err == nil && ok {
		switch res.DirectClimatisation.ClimatisationState {
		case "OFF":
			return false, 0, 0, nil
		case "ON":
			return true, 0, 0, nil
		}
	}

	return active, outsideTemp, targetTemp, err
}
