package vw

import (
	"math"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// Provider implements the evcc vehicle api
type Provider struct {
	chargerG          func() (interface{}, error)
	climateG          func() (interface{}, error)
	startChargeAction func() error
}

// NewProvider provides the evcc vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		chargerG: provider.NewCached(func() (interface{}, error) {
			return api.Charger(vin)
		}, cache).InterfaceGetter(),
		climateG: provider.NewCached(func() (interface{}, error) {
			return api.Climater(vin)
		}, cache).InterfaceGetter(),
		startChargeAction: func() error {
			return api.Action(vin, ActionCharge, ActionChargeStart)
		},
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.chargerG()
	if res, ok := res.(ChargerResponse); err == nil && ok {
		return float64(res.Charger.Status.BatteryStatusData.StateOfCharge.Content), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
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

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.chargerG()
	if res, ok := res.(ChargerResponse); err == nil && ok {
		rct := res.Charger.Status.BatteryStatusData.RemainingChargingTime

		// estimate not available
		if rct.Content == 65535 {
			return time.Time{}, api.ErrNotAvailable
		}

		timestamp, err := time.Parse(time.RFC3339, rct.Timestamp)
		return timestamp.Add(time.Duration(rct.Content) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (rng int64, err error) {
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

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	res, err := v.climateG()
	if res, ok := res.(ClimaterResponse); err == nil && ok {
		state := strings.ToLower(res.Climater.Status.ClimatisationStatusData.ClimatisationState.Content)
		active := state != "off" && state != "invalid" && state != "error"

		targetTemp = res.Climater.Settings.TargetTemperature.Content
		outsideTemp = res.Climater.Status.TemperatureStatusData.OutdoorTemperature.Content
		if math.IsNaN(outsideTemp) {
			outsideTemp = targetTemp // cover "invalid"
		}

		return active, outsideTemp, targetTemp, nil
	}

	return active, outsideTemp, targetTemp, err
}

var _ api.VehicleStartCharge = (*Provider)(nil)

// StartCharge implements the api.VehicleStartCharge interface
func (v *Provider) StartCharge() error {
	return v.startChargeAction()
}
