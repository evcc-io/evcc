package ford

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

const refreshTimeout = time.Minute

type Provider struct {
	statusG     func() (StatusResponse, error)
	expiry      time.Duration
	refreshTime time.Time
	refreshId   string
	wakeup      func() error
}

func NewProvider(api *API, vin string, expiry, cache time.Duration) *Provider {
	impl := &Provider{
		expiry: expiry,
	}

	impl.statusG = provider.Cached(func() (StatusResponse, error) {
		return impl.status(
			func() (StatusResponse, error) { return api.Status(vin) },
			func(id string) (StatusResponse, error) { return api.RefreshResult(vin, id) },
			func() (string, error) { return api.RefreshRequest(vin) },
		)
	}, cache)

	impl.wakeup = func() error { return api.WakeUp(vin) }

	return impl
}

func (v *Provider) status(
	statusG func() (StatusResponse, error),
	refreshG func(id string) (StatusResponse, error),
	refreshRequest func() (string, error),
) (StatusResponse, error) {
	if v.refreshId != "" {
		res, err := refreshG(v.refreshId)

		// update successful and completed
		if err == nil {
			v.refreshId = ""
			return res, nil
		}

		// update still in progress, keep retrying
		if time.Since(v.refreshTime) < refreshTimeout {
			return res, api.ErrMustRetry
		}

		// give up
		v.refreshId = ""
		return res, api.ErrTimeout
	}

	res, err := statusG()

	if err == nil {
		if time.Since(res.VehicleStatus.LastRefresh.Time) > v.expiry {
			if v.refreshId, err = refreshRequest(); err == nil {
				v.refreshTime = time.Now()
				err = api.ErrMustRetry
			}
		}
	}

	return res, err
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Battery interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	if err == nil {
		return res.VehicleStatus.BatteryFillLevel.Value, nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err == nil {
		return int64(res.VehicleStatus.ElVehDTE.Value), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if err == nil {
		if res.VehicleStatus.PlugStatus.Value == 1 {
			status = api.StatusB // connected, not charging
		}
		if res.VehicleStatus.ChargingStatus.Value == "ChargingAC" {
			status = api.StatusC // charging
		}
	}

	return status, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	return res.VehicleStatus.Odometer.Value, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.statusG()
	return res.VehicleStatus.Gps.Latitude, res.VehicleStatus.Gps.Longitude, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()
	return res.VehicleStatus.ChargeEndTime.Value.Time, err
}

var _ api.Resurrector = (*Provider)(nil)

// WakeUp implements the api.Resurrector interface
func (v *Provider) WakeUp() error {
	return v.wakeup()
}
