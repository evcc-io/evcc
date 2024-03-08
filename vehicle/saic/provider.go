package saic

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/vehicle/saic/requests"
)

const (
	Target40  = 1
	Target50  = 2
	Target60  = 3
	Target70  = 4
	Target80  = 5
	Target90  = 6
	Target100 = 7
)

var TargetSocVals = [...]int{0, 40, 50, 60, 70, 80, 90, 100}

// https://github.com/SAIC-iSmart-API/reverse-engineering

// Provider implements the vehicle api
type Provider struct {
	status provider.Cacheable[requests.ChargeStatus]
	wakeup func() error
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		status: provider.ResettableCached(func() (requests.ChargeStatus, error) {
			return api.Status(vin)
		}, cache),
		wakeup: func() error {
			return api.Wakeup(vin)
		},
	}

	return impl
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.status.Get()
	if err != nil {
		return 0, err
	}

	val := res.ChrgMgmtData.BmsPackSOCDsp
	if val > 1000 {
		v.status.Reset()
		return float64(val), fmt.Errorf("invalid raw soc value: %d: %w", val, api.ErrMustRetry)
	}

	return float64(val) / 10.0, nil
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.status.Get()
	if err != nil {
		return api.StatusNone, err
	}

	status := api.StatusA // disconnected
	if res.RvsChargeStatus.ChargingGunState != 0 {
		if (res.ChrgMgmtData.BmsChrgSts & 0x01) == 0 {
			status = api.StatusB
		} else {
			status = api.StatusC
		}
	}

	return status, nil
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.status.Get()
	if err == nil {
		ctr := res.ChrgMgmtData.ChrgngRmnngTime
		return time.Now().Add(time.Duration(ctr) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.status.Get()
	if err != nil {
		return 0, err
	}
	val := res.RvsChargeStatus.FuelRangeElec
	if val < 10 {
		// Ok, 0 would be possible, but it's more likely that it's an invalid answer.
		return val, fmt.Errorf("invalid raw range value: %d: %w", val, api.ErrMustRetry)
	}
	return val / 10, nil
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.status.Get()
	if err != nil {
		return 0, err
	}

	return float64(res.RvsChargeStatus.Mileage) / 10.0, nil
}

var _ api.SocLimiter = (*Provider)(nil)

// TargetSoc implements the api.SocLimiter interface
func (v *Provider) TargetSoc() (float64, error) {
	var result = 0
	res, err := v.status.Get()
	if err != nil {
		return 0, err
	}

	index := res.ChrgMgmtData.BmsOnBdChrgTrgtSOCDspCmd

	if index <= Target100 {
		result = TargetSocVals[res.ChrgMgmtData.BmsOnBdChrgTrgtSOCDspCmd]
	}

	return float64(result), err
}

var _ api.Resurrector = (*Provider)(nil)

func (v *Provider) WakeUp() error {
	return v.wakeup()
}
