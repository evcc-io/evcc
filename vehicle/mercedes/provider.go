package mercedes

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

type Provider struct {
	dataG func() (StatusResponse, error)
	log   *util.Logger
}

func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		dataG: provider.Cached(func() (StatusResponse, error) {
			res, err := api.Status(vin)
			return res, err
		}, cache),
		log: api.log,
	}
	return impl
}

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.dataG()
	if err == nil {
		return res.EvInfo.Battery.StateOfCharge, nil
	}

	v.log.DEBUG.Printf("Provider: SOC - %s", err)
	return 0, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.dataG()
	if err == nil {
		return int64(res.EvInfo.Battery.DistanceToEmpty.Value), nil
	}

	v.log.DEBUG.Printf("Provider: Range - %s", err)
	return 0, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.dataG()
	if err == nil {
		return float64(res.VehicleInfo.Odometer.Value), nil
	}

	v.log.DEBUG.Printf("Provider: Odo - %s", err)
	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.dataG()
	if err == nil {
		status = MapChargeStatus(res.EvInfo.Battery.ChargingStatus)
	}

	v.log.DEBUG.Printf("Provider: Status - %s", fmt.Sprint(res.EvInfo.Battery.ChargingStatus))
	return status, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.dataG()
	if err == nil {
		return res.LocationResponse.Latitude, res.LocationResponse.Longitude, nil
	}

	v.log.DEBUG.Printf("Provider: Loc - %s", err)
	return 0, 0, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.dataG()
	if err != nil {
		return time.Time{}, err
	}
	v.log.DEBUG.Printf("Provider: FinishTime - %s", fmt.Sprint(res.EvInfo.Battery.EndOfChargeTime))

	now := time.Now()
	targetDateTime := time.Date(now.Year(), now.Month(), now.Day(), 0, res.EvInfo.Battery.EndOfChargeTime, 0, 0, now.Location())

	if targetDateTime.Before(now) {
		targetDateTime = targetDateTime.Add(24 * time.Hour)
	}
	return targetDateTime, nil
}

// Charging Status
// 0=CHARGING
// 1=CHARGING_ENDS
// 2=CHARGE_BREAK
// 3=UNPLUGGED
// 4=FAILURE
// 5=SLOW
// 6=FAST
// 7=DISCHARGING
// 8=NO_CHARGING
// 9=SLOW_CHARGING_AFTER_REACHING_TRIP_TARGET
// 10=CHARGING_AFTER_REACHING_TRIP_TARGET
// 11=FAST_CHARGING_AFTER_REACHING_TRIP_TARGET
// 12=UNKNOWN

func MapChargeStatus(lookup int) api.ChargeStatus {
	switch lookup {
	case
		0, 5, 6, 9, 10, 11:
		return api.StatusC
	case
		1, 2, 4, 7, 8:
		return api.StatusB
	}
	return api.StatusA
}
