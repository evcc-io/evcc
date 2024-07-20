package zero

import (
	"fmt"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

// Provider implements the vehicle api
type Provider struct {
	api    *API
	status provider.Cacheable[ZeroState]
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, cache time.Duration) *Provider {
	impl := &Provider{
		api: api,
		status: provider.ResettableCached(func() (ZeroState, error) {
			return api.Status()
		}, cache),
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

	val := res.Soc

	return float64(val), nil
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.status.Get()
	if err != nil {
		return api.StatusNone, err
	}

	status := api.StatusA // disconnected

	if res.Pluggedin != 0 {
		if (res.Charging) == 0 {
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
	var t time.Time
	res, err := v.status.Get()
	if err == nil {
		if len(res.Datetime_actual) == 14 {
			convTime := fmt.Sprintf("%s-%s-%s %s:%s:%s",
				res.Datetime_actual[0:4], res.Datetime_actual[4:6], res.Datetime_actual[6:8],
				res.Datetime_actual[8:10], res.Datetime_actual[10:12], res.Datetime_actual[12:14])

			// 2023-11-14 13:23:45
			t, err = time.Parse(time.DateTime, convTime)

		}
		if err != nil {
			t = time.Now()
		}

		ctr := res.Chargingtimeleft
		return t.Add(time.Duration(ctr) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.status.Get()
	if err != nil {
		return 0, err
	}

	var mileage float64
	mileage, err = strconv.ParseFloat(res.Mileage, 64)
	if err != nil {
		return 0, err
	}

	return mileage, nil
}
