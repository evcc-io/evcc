package skoda

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

// Provider implements the evcc vehicle api
type Provider struct {
	statusG  func() (StatusResponse, error)
	chargerG func() (ChargerResponse, error)
	action   func(action, value string) error
}

// NewProvider provides the evcc vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.Cached(func() (StatusResponse, error) {
			return api.Status(vin)
		}, cache),
		chargerG: provider.Cached(func() (ChargerResponse, error) {
			return api.Charger(vin)
		}, cache),
		// climateG: provider.Cached(func() (interface{}, error) {
		// 	return api.Climater(vin)
		// }, cache),
		action: func(action, value string) error {
			return api.Action(vin, action, value)
		},
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.chargerG()
	if err == nil {
		return float64(res.Battery.StateOfChargeInPercent), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.chargerG()
	if err == nil {
		if res.Plug.ConnectionState == "Connected" {
			status = api.StatusB
		}
		if res.Charging.State == "Charging" {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.chargerG()
	if err == nil {
		crg := res.Charging

		// estimate not available
		if crg.State == "Error" || crg.ChargingType == "Invalid" {
			return time.Time{}, api.ErrNotAvailable
		}

		remaining := time.Duration(crg.RemainingToCompleteInSeconds) * time.Second
		return time.Now().Add(remaining), err
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (rng int64, err error) {
	res, err := v.chargerG()
	return res.Battery.CruisingRangeElectricInMeters / 1e3, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (odo float64, err error) {
	res, err := v.statusG()
	return res.Remote.MileageInKm, err
}

// var _ api.VehicleClimater = (*Provider)(nil)

// // Climater implements the api.VehicleClimater interface
// func (v *Provider) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
// 	res, err := v.climateG()
// err == nil {
// 		state := strings.ToLower(res.Climater.Status.ClimatisationStatusData.ClimatisationState.Content)
// 		active := state != "off" && state != "invalid" && state != "error"

// 		targetTemp = res.Climater.Settings.TargetTemperature.Content
// 		outsideTemp = res.Climater.Status.TemperatureStatusData.OutdoorTemperature.Content
// 		if math.IsNaN(outsideTemp) {
// 			outsideTemp = targetTemp // cover "invalid"
// 		}

// 		return active, outsideTemp, targetTemp, nil
// 	}

// 	return active, outsideTemp, targetTemp, err
// }

var _ api.VehicleChargeController = (*Provider)(nil)

// StartCharge implements the api.VehicleChargeController interface
func (v *Provider) StartCharge() error {
	return v.action(ActionCharge, ActionChargeStart)
}

// StopCharge implements the api.VehicleChargeController interface
func (v *Provider) StopCharge() error {
	return v.action(ActionCharge, ActionChargeStop)
}
