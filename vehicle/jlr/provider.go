package jlr

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

type Provider struct {
	statusG   func() (interface{}, error)
	positionG func() (interface{}, error)
	actionS   func(bool) error
}

func NewProvider(api *API, vin, user string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.NewCached(func() (interface{}, error) {
			return api.Status(vin)
		}, cache).InterfaceGetter(),
		positionG: provider.NewCached(func() (interface{}, error) {
			return api.Position(vin)
		}, cache).InterfaceGetter(),
		actionS: func(start bool) error {
			return api.ChargeAction(vin, user, start)
		},
	}

	return impl
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Battery interface
func (v *Provider) SoC() (float64, error) {
	var val float64
	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		val, err = res.VehicleStatus.EvStatus.FloatVal("EV_STATE_OF_CHARGE")
	}

	return val, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	var val int64
	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		val, err = res.VehicleStatus.EvStatus.IntVal("EV_RANGE_ON_BATTERY_KM")
	}

	return val, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		if s, err := res.VehicleStatus.EvStatus.StringVal("EV_CHARGING_STATUS"); err == nil {
			switch s {
			case "NOTCONNECTED":
				status = api.StatusA
			case "INITIALIZATION", "PAUSED":
				status = api.StatusB
			case "CHARGING":
				status = api.StatusC
			}
		}
	}

	return status, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		i, err := res.VehicleStatus.CoreStatus.IntVal("EV_MINUTES_TO_FULLY_CHARGED")
		return time.Now().Add(time.Duration(i) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	var val float64
	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		val, err = res.VehicleStatus.CoreStatus.FloatVal("ODOMETER")
	}

	return val / 1e3, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.positionG()
	if res, ok := res.(PositionResponse); err == nil && ok {
		return res.Position.Latitude, res.Position.Longitude, nil
	}

	return 0, 0, err
}

var _ api.VehicleStartCharge = (*Provider)(nil)

// StartCharge implements the api.VehicleStartCharge interface
func (v *Provider) StartCharge() error {
	return v.actionS(true)
}

var _ api.VehicleStopCharge = (*Provider)(nil)

// StopCharge implements the api.VehicleStopCharge interface
func (v *Provider) StopCharge() error {
	return v.actionS(false)
}
