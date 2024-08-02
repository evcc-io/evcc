package bmw

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

// Provider implements the vehicle api
type Provider struct {
	statusG func() (VehicleStatus, error)
	actionS func(action string) error
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.Cached(func() (VehicleStatus, error) {
			return api.Status(vin)
		}, cache),
		actionS: func(action string) error {
			_, err := api.Action(vin, action)
			return err
		},
	}
	return impl
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}

	return float64(res.State.ElectricChargingState.ChargingLevelPercent), nil
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.statusG()
	if err != nil {
		return api.StatusNone, err
	}

	status := api.StatusA // disconnected
	if res.State.ElectricChargingState.IsChargerConnected {
		status = api.StatusB
	}
	if res.State.ElectricChargingState.ChargingStatus == "CHARGING" {
		status = api.StatusC
	}

	return status, nil
}

// var _ api.VehicleFinishTimer = (*Provider)(nil)

// // FinishTime implements the api.VehicleFinishTimer interface
// func (v *Provider) FinishTime() (time.Time, error) {
// 	res, err := v.statusG()
// err == nil {
// 		ctr := res.VehicleStatus.ChargingTimeRemaining
// 		return time.Now().Add(time.Duration(ctr) * time.Minute), err
// 	}

// 	return time.Time{}, err
// }

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}

	return res.State.ElectricChargingState.Range, nil
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}

	return float64(res.State.CurrentMileage), nil
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}

	return res.State.ElectricChargingState.ChargingTarget, nil
}

var _ api.Resurrector = (*Provider)(nil)

func (v *Provider) WakeUp() error {
	return v.actionS(DOOR_LOCK)
}

var _ api.ChargeController = (*Provider)(nil)

// ChargeEnable implements the api.ChargeController interface
func (v *Provider) ChargeEnable(enable bool) error {
	action := map[bool]string{true: CHARGE_START, false: CHARGE_STOP}
	return v.actionS(action[enable])
}
