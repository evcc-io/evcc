package zero

import (
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

// Provider implements the vehicle api
type Provider struct {
	status provider.Cacheable[ZeroState]
}

// NewProvider creates a vehicle api provider
func NewProvider(api *API, cache time.Duration) *Provider {
	impl := &Provider{
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
	res, err := v.status.Get()
	if err == nil {
		ctr := res.Chargingtimeleft
		return time.Now().Add(time.Duration(ctr) * time.Minute), err
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

	var mileage int
	mileage, err = strconv.Atoi(res.Mileage)
	if err != nil {
		return 0, err
	}

	return float64(mileage), nil
}
