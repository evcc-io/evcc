package vw

import (
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// Implementation implements the evcc vehicle api
type Implementation struct {
	chargerG func() (interface{}, error)
	climateG func() (interface{}, error)
}

// NewImplementation provides the evcc vehicle api implementation
func NewImplementation(api *API, cache time.Duration) *Implementation {
	impl := &Implementation{
		chargerG: provider.NewCached(func() (interface{}, error) {
			return api.Charger()
		}, cache).InterfaceGetter(),
		climateG: provider.NewCached(func() (interface{}, error) {
			return api.Climater()
		}, cache).InterfaceGetter(),
	}
	return impl
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Implementation) ChargeState() (float64, error) {
	res, err := v.chargerG()
	if res, ok := res.(ChargerResponse); err == nil && ok {
		return float64(res.Charger.Status.BatteryStatusData.StateOfCharge.Content), nil
	}

	return 0, err
}

// FinishTime implements the Vehicle.ChargeFinishTimer interface
func (v *Implementation) FinishTime() (time.Time, error) {
	res, err := v.chargerG()
	if res, ok := res.(ChargerResponse); err == nil && ok {
		var timestamp time.Time
		if err == nil {
			timestamp, err = time.Parse(time.RFC3339, res.Charger.Status.BatteryStatusData.RemainingChargingTime.Timestamp)
		}

		return timestamp.Add(time.Duration(res.Charger.Status.BatteryStatusData.RemainingChargingTime.Content) * time.Minute), err
	}

	return time.Time{}, err
}

// Status implements the Vehicle.Status interface
func (v *Implementation) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.chargerG()
	if res, ok := res.(ChargerResponse); err == nil && ok {
		if res.Charger.Status.PlugStatusData.PlugState.Content == "connected" {
			status = api.StatusB
		}
		if res.Charger.Status.ChargingStatusData.ChargingState.Content == "charging" {
			status = api.StatusC
		}
	}

	return status, err
}

// Climater implements the Vehicle.Climater interface
func (v *Implementation) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	res, err := v.climateG()
	if res, ok := res.(ClimaterResponse); err == nil && ok {
		active = "off" != strings.ToLower(res.Climater.Status.ClimatisationStatusData.ClimatisationState.Content)
		outsideTemp = Temp2Float(res.Climater.Status.TemperatureStatusData.OutdoorTemperature.Content)
		targetTemp = Temp2Float(res.Climater.Settings.TargetTemperature.Content)

		return active, outsideTemp, targetTemp, nil
	}

	return active, outsideTemp, targetTemp, err
}
