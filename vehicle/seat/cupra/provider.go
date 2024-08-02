package cupra

import (
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

// Provider is an api.Vehicle implementation for Seat Cupra cars
type Provider struct {
	statusG   func() (Status, error)
	positionG func() (Position, error)
	milageG   func() (Mileage, error)
	action    func(string, string) error
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, userID, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.Cached(func() (Status, error) {
			return api.Status(userID, vin)
		}, cache),
		positionG: provider.Cached(func() (Position, error) {
			return api.ParkingPosition(vin)
		}, cache),
		milageG: provider.Cached(func() (Mileage, error) {
			return api.Mileage(vin)
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
	return res.Engines.Primary.LevelPct, err
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
	if err != nil {
		return time.Time{}, err
	}

	rsc := res.Services.Charging
	if !rsc.Active {
		return time.Time{}, api.ErrNotAvailable
	}

	rt := rsc.RemainingTime
	if rsc.TargetPct > 0 && rsc.TargetPct < 100 {
		rt = rt * 100 / int64(rsc.TargetPct)
	}

	return time.Now().Add(time.Duration(rt) * time.Minute), nil
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	return int64(res.Engines.Primary.RangeKm), err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.milageG()
	return res.MileageKm, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.positionG()
	return res.Lat, res.Lon, err
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res, err := v.statusG()
	return res.Services.Climatisation.Active, err
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.statusG()
	return int64(res.Services.Charging.TargetPct), err
}

var _ api.ChargeController = (*Provider)(nil)

// ChargeEnable implements the api.ChargeController interface
func (v *Provider) ChargeEnable(enable bool) error {
	action := map[bool]string{true: ActionChargeStart, false: ActionChargeStop}
	return v.action(ActionCharge, action[enable])
}
