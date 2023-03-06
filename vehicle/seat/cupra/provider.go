package cupra

import (
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

// Provider is an api.Vehicle implementation for Seat Cupra cars
type Provider struct {
	statusG func() (Status, error)
	action  func(string, string) error
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, userID, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.Cached(func() (Status, error) {
			return api.Status(userID, vin)
		}, cache),
		action: func(action, cmd string) error {
			return api.Action(vin, action, cmd)
		},
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	return float64(res.Engines.Primary.Level), err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if err == nil {
		switch strings.ToLower(res.Services.Charging.Status) {
		case "connected", "readyforcharging":
			status = api.StatusB
		case "charging":
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if err == nil {
		rsc := res.Services.Charging
		if !rsc.Active {
			return time.Time{}, api.ErrNotAvailable
		}

		rt := rsc.RemainingTime
		if rsc.TargetPct > 0 && rsc.TargetPct < 100 {
			rt = rt * 100 / int64(rsc.TargetPct)
		}

		return time.Now().Add(time.Duration(rt) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	return int64(res.Engines.Primary.Range.Value), err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res, err := v.statusG()
	return res.Services.Climatisation.Active, err
}

var _ api.SocLimiter = (*Provider)(nil)

// TargetSoc implements the api.SocLimiter interface
func (v *Provider) TargetSoc() (float64, error) {
	res, err := v.statusG()
	if err == nil {
		return float64(res.Services.Charging.TargetPct), nil
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
