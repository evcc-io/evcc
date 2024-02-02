package saic

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/vehicle/saic/requests"
)

// https://github.com/SAIC-iSmart-API/reverse-engineering

// Provider implements the vehicle api
type Provider struct {
	statusG func() (requests.ChargeStatus, error)
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.Cached(func() (requests.ChargeStatus, error) {
			return api.Status(vin)
		}, cache),
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

	return float64(res.ChrgMgmtData.BmsPackSOCDsp) / 10.0, nil
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.statusG()
	if err != nil {
		return api.StatusNone, err
	}

	status := api.StatusA // disconnected
	if res.RvsChargeStatus.ChargingGunState != 0 {
		status = api.StatusB
	}
	if res.RvsChargeStatus.ChargingType != 0 {
		status = api.StatusC
	}

	return status, nil
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if err == nil {
		ctr := res.ChrgMgmtData.ChrgngRmnngTime
		return time.Now().Add(time.Duration(ctr) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}

	return res.RvsChargeStatus.FuelRangeElec / 10, nil
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}

	return float64(res.RvsChargeStatus.Mileage), nil
}

var _ api.Resurrector = (*Provider)(nil)

func (v *Provider) WakeUp() error {
	_, err := v.statusG()
	return err
}

/*
var _ api.VehicleChargeController = (*Provider)(nil)

// StartCharge implements the api.VehicleChargeController interface
func (v *Provider) StartCharge() error {
	return v.actionS(CHARGE_START)
}

// StopCharge implements the api.VehicleChargeController interface
func (v *Provider) StopCharge() error {
	return v.actionS(CHARGE_STOP)
}
*/
