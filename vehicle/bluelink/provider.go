package bluelink

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

const refreshTimeout = 2 * time.Minute

// Provider implements the vehicle api.
// Based on https://github.com/Hacksore/bluelinky.
type Provider struct {
	statusG     func() (BluelinkVehicleStatus, error)
	statusLG    func() (BluelinkVehicleStatusLatest, error)
	refreshG    func() (BluelinkVehicleStatus, error)
	expiry      time.Duration
	refreshTime time.Time
}

// New creates a new BlueLink API
func NewProvider(api *API, vehicle Vehicle, expiry, cache time.Duration) *Provider {
	v := &Provider{
		refreshG: func() (BluelinkVehicleStatus, error) {
			return api.StatusPartial(vehicle)
		},
		expiry: expiry,
	}

	v.statusG = provider.Cached(func() (BluelinkVehicleStatus, error) {
		return v.status(
			func() (BluelinkVehicleStatusLatest, error) { return api.StatusLatest(vehicle) },
		)
	}, cache)

	v.statusLG = provider.Cached(func() (BluelinkVehicleStatusLatest, error) {
		return api.StatusLatest(vehicle)
	}, cache)

	return v
}

// status wraps the api status call and adds status refresh
func (v *Provider) status(statusG func() (BluelinkVehicleStatusLatest, error)) (BluelinkVehicleStatus, error) {
	res, err := statusG()

	var ts time.Time
	if err == nil {
		ts, err = res.BluelinkVehicleStatus().Updated()
		if err != nil {
			return res.BluelinkVehicleStatus(), err
		}

		// return the current value
		if time.Since(ts) <= v.expiry {
			v.refreshTime = time.Time{}
			return res.BluelinkVehicleStatus(), nil
		}
	}

	// request a refresh, irrespective of a previous error
	if v.refreshTime.IsZero() {
		v.refreshTime = time.Now()

		// TODO async refresh
		res, err := v.refreshG()
		if err == nil {
			if ts, err = res.Updated(); err == nil && time.Since(ts) <= v.expiry {
				v.refreshTime = time.Time{}
				return res, nil
			}

			err = api.ErrMustRetry
		}

		return nil, err
	}

	// refresh finally expired
	if time.Since(v.refreshTime) > refreshTimeout {
		v.refreshTime = time.Time{}
		if err == nil {
			err = api.ErrTimeout
		}
	} else {
		// wait for refresh, irrespective of a previous error
		err = api.ErrMustRetry
	}

	return nil, err
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}
	return res.SoC()
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.Battery interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusNone
	res, err := v.statusG()
	if err != nil {
		return status, err
	}
	return res.Status()
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	if err != nil {
		return time.Time{}, err
	}
	return res.FinishTime()
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}
	return res.Range()
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusLG()
	if err != nil {
		return 0, err
	}
	return res.Odometer()
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}
	return res.GetLimitSoc()
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.statusLG()
	if err != nil {
		return 0, 0, err
	}
	return res.Position()
}

var _ api.Resurrector = (*Provider)(nil)

// WakeUp implements the api.Resurrector interface
func (v *Provider) WakeUp() error {
	// forcing an update will usually make the car start charging even if the (first) resulting status still says it does not charge...
	_, err := v.refreshG()
	return err
}
