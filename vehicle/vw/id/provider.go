package id

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/provider"
)

// Provider is an api.Vehicle implementation for VW ID cars
type Provider struct {
	statusG        func() (Status, error)
	action         func(action, value string) error
	maxChargeLevel func(value string) error
	lp             loadpoint.API
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.Cached(func() (Status, error) {
			return api.Status(vin)
		}, cache),
		action: func(action, value string) error {
			return api.Action(vin, action, value)
		},
		maxChargeLevel: func(value string) error {
			return api.MaxChargeLevel(vin, value)
		},
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()

	var eng EngineRangeStatus
	if err == nil {
		eng, err = res.FuelStatus.EngineRangeStatus("electric")
	}

	if err == nil {
		return float64(eng.CurrentSOCPct), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.statusG()
	if err == nil && res.Charging == nil {
		err = errors.New("missing charging status")
	}

	status := api.StatusA // disconnected
	if err == nil {
		if res.Charging.PlugStatus.Value.PlugConnectionState == "connected" {
			status = api.StatusB
		}
		if res.Charging.ChargingStatus.Value.ChargingState == "charging" {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if err == nil && res.Charging == nil {
		err = errors.New("missing charging status")
	}

	if err == nil {
		cst := res.Charging.ChargingStatus.Value
		return cst.CarCapturedTimestamp.Add(time.Duration(cst.RemainingChargingTimeToCompleteMin) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err == nil && res.FuelStatus == nil {
		err = api.ErrNotAvailable
	}

	var eng EngineRangeStatus
	if err == nil {
		eng, err = res.FuelStatus.EngineRangeStatus("electric")
	}

	if err == nil {
		return int64(eng.RemainingRangeKm), nil
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	if err == nil && res.Measurements == nil {
		err = api.ErrNotAvailable
	}

	if err == nil {
		return res.Measurements.OdometerStatus.Value.Odometer, nil
	}

	return 0, err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res, err := v.statusG()
	if err == nil && res.Climatisation == nil {
		err = api.ErrNotAvailable
	}

	if err == nil {
		state := strings.ToLower(res.Climatisation.ClimatisationStatus.Value.ClimatisationState)
		if state == "" {
			return false, api.ErrNotAvailable
		}

		active := state != "off" && state != "invalid" && state != "error"
		return active, nil
	}

	return false, err
}

var _ api.SocLimiter = (*Provider)(nil)

// TargetSoc implements the api.SocLimiter interface
func (v *Provider) TargetSoc() (float64, error) {
	res, err := v.statusG()
	if err == nil && res.Charging == nil {
		err = errors.New("missing charging status")
	}

	if err == nil {
		return float64(res.Charging.ChargingSettings.Value.TargetSOCPct), nil
	}

	return 0, err
}

var _ api.VehicleChargeController = (*Provider)(nil)

// StartCharge implements the api.VehicleChargeController interface
func (v *Provider) StartCharge() error {
	return v.action(ActionCharge, ActionChargeStart)
}

// StopCharge implements the api.VehicleChargeController interface
func (v *Provider) StopCharge() error {
	return v.action(ActionCharge, ActionChargeStop)
}

var _ api.Resurrector = (*Provider)(nil)

// WakeUp implements the api.Resurrector interface
func (v *Provider) WakeUp() error {
	return v.StartCharge()
}

// MaxCurrent implements the api.Charger interface
func (v *Provider) MaxCurrent(current int64) error {
	fmt.Printf("current: %d\n", current)
	fmt.Printf("lp: %+v\n", v.lp)

	// commented out to avoid spamming API calls during development

	// e-up: there are only two possible levels
	// "reduced" or "maximum"; reduced is 5A
	// level := "maximum"
	// if current <= 5 {
	// 	level = "reduced"
	// }

	//return v.maxChargeLevel(level)
	return nil
}

var _ loadpoint.Controller = (*Provider)(nil)

// LoadpointControl implements loadpoint.Controller
func (v *Provider) LoadpointControl(lp loadpoint.API) {
	v.lp = lp
}
