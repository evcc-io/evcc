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
	statusG     func() (VehicleStatus, error)
	statusLG    func() (StatusLatestResponse, error)
	refreshG    func() (StatusResponse, error)
	expiry      time.Duration
	refreshTime time.Time
}

// New creates a new BlueLink API
func NewProvider(api *API, vid string, expiry, cache time.Duration) *Provider {
	v := &Provider{
		refreshG: func() (StatusResponse, error) {
			return api.StatusPartial(vid)
		},
		expiry: expiry,
	}

	v.statusG = provider.Cached(func() (VehicleStatus, error) {
		return v.status(
			func() (StatusLatestResponse, error) { return api.StatusLatest(vid) },
		)
	}, cache)

	v.statusLG = provider.Cached(func() (StatusLatestResponse, error) {
		return api.StatusLatest(vid)
	}, cache)

	return v
}

// status wraps the api status call and adds status refresh
func (v *Provider) status(statusG func() (StatusLatestResponse, error)) (VehicleStatus, error) {
	res, err := statusG()

	var ts time.Time
	if err == nil {
		ts, err = res.ResMsg.VehicleStatusInfo.VehicleStatus.Updated()
		if err != nil {
			return res.ResMsg.VehicleStatusInfo.VehicleStatus, err
		}

		// return the current value
		if time.Since(ts) <= v.expiry {
			v.refreshTime = time.Time{}
			return res.ResMsg.VehicleStatusInfo.VehicleStatus, nil
		}
	}

	// request a refresh, irrespective of a previous error
	if v.refreshTime.IsZero() {
		v.refreshTime = time.Now()

		// TODO async refresh
		res, err := v.refreshG()
		if err == nil {
			if ts, err = res.ResMsg.Updated(); err == nil && time.Since(ts) <= v.expiry {
				v.refreshTime = time.Time{}
				return res.ResMsg, nil
			}

			err = api.ErrMustRetry
		}

		return VehicleStatus{}, err
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

	return VehicleStatus{}, err
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()

	if err == nil {
		return res.EvStatus.BatteryStatus, nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.Battery interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.statusG()

	status := api.StatusNone
	if err == nil {
		status = api.StatusA
		if res.EvStatus.BatteryPlugin > 0 || res.EvStatus.ChargePortDoorOpenStatus == 1 {
			status = api.StatusB
		}
		if res.EvStatus.BatteryCharge {
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
		remaining := res.EvStatus.RemainTime2.Atc.Value

		if remaining == 0 {
			return time.Time{}, api.ErrNotAvailable
		}

		ts, err := res.Updated()
		return ts.Add(time.Duration(remaining) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()

	if err == nil {
		if dist := res.EvStatus.DrvDistance; len(dist) == 1 {
			return int64(dist[0].RangeByFuel.EvModeRange.Value), nil
		}

		return 0, api.ErrNotAvailable
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusLG()
	return res.ResMsg.VehicleStatusInfo.Odometer.Value, err
}

var _ api.SocLimiter = (*Provider)(nil)

// TargetSoc implements the api.SocLimiter interface
func (v *Provider) TargetSoc() (float64, error) {
	res, err := v.statusG()

	if err == nil {
		for _, targetSOC := range res.EvStatus.ReservChargeInfos.TargetSocList {
			if targetSOC.PlugType == plugTypeAC {
				return float64(targetSOC.TargetSocLevel), nil
			}
		}
	}

	return 0, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.statusLG()
	coord := res.ResMsg.VehicleStatusInfo.VehicleLocation.Coord
	return coord.Lat, coord.Lon, err
}

var _ api.Resurrector = (*Provider)(nil)

// WakeUp implements the api.Resurrector interface
func (v *Provider) WakeUp() error {
	// forcing an update will usually make the car start charging even if the (first) resulting status still says it does not charge...
	_, err := v.refreshG()
	return err
}
