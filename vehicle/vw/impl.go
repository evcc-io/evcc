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
		rct := res.Charger.Status.BatteryStatusData.RemainingChargingTime

		// estimate not available
		if rct.Content == 65535 {
			return time.Time{}, api.ErrNotAvailable
		}

		var timestamp time.Time
		timestamp, err = time.Parse(time.RFC3339, rct.Timestamp)

		return timestamp.Add(time.Duration(rct.Content) * time.Minute), err
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

// Range implements the Vehicle.Range interface
func (v *Implementation) Range() (rng int64, err error) {
	res, err := v.chargerG()
	if res, ok := res.(ChargerResponse); err == nil && ok {
		crsd := res.Charger.Status.CruisingRangeStatusData

		rng = int64(crsd.PrimaryEngineRange.Content)
		if crsd.EngineTypeFirstEngine.Content != "typeIsElectric" {
			rng = int64(crsd.SecondaryEngineRange.Content)
		}
	}

	return rng, err
}

// Climater implements the Vehicle.Climater interface
func (v *Implementation) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	res, err := v.climateG()
	if res, ok := res.(ClimaterResponse); err == nil && ok {
		state := strings.ToLower(res.Climater.Status.ClimatisationStatusData.ClimatisationState.Content)
		active := state != "off" && state != "invalid" && state != "error"

		outsideTemp = Temp2Float(res.Climater.Status.TemperatureStatusData.OutdoorTemperature.Content)
		targetTemp = Temp2Float(res.Climater.Settings.TargetTemperature.Content)

		return active, outsideTemp, targetTemp, nil
	}

	return active, outsideTemp, targetTemp, err
}
