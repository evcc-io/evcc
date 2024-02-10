package nissan

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

const refreshTimeout = 2 * time.Minute

// Provider is a kamereon provider
type Provider struct {
	statusG     func() (StatusResponse, error)
	action      func(value Action) error
	expiry      time.Duration
	refreshTime time.Time
}

// NewProvider returns a kamereon provider
func NewProvider(api *API, vin string, expiry, cache time.Duration) *Provider {
	impl := &Provider{
		action: func(value Action) error {
			_, err := api.ChargingAction(vin, value)
			return err
		},
		expiry: expiry,
	}

	impl.statusG = provider.Cached(func() (StatusResponse, error) {
		return impl.status(
			func() (StatusResponse, error) { return api.BatteryStatus(vin) },
			func() (ActionResponse, error) { return api.RefreshRequest(vin, "RefreshBatteryStatus") },
		)
	}, cache)

	return impl
}

func (v *Provider) status(battery func() (StatusResponse, error), refresh func() (ActionResponse, error)) (StatusResponse, error) {
	res, err := battery()

	if err == nil {
		// result valid?
		if time.Since(res.Attributes.LastUpdateTime.Time) < v.expiry || res.Attributes.LastUpdateTime.IsZero() {
			v.refreshTime = time.Time{}
			return res, err
		}
	}

	// request a refresh, irrespective of a previous error
	if v.refreshTime.IsZero() {
		if _, err = refresh(); err == nil {
			v.refreshTime = time.Now()
			err = api.ErrMustRetry
		}

		return res, err
	}

	// refresh finally expired
	if time.Since(v.refreshTime) > refreshTimeout {
		v.refreshTime = time.Time{}
		if err == nil {
			err = api.ErrTimeout
		}
	} else {
		if len(res.Errors) > 0 {
			// extract error code
			e := res.Errors[0]
			err = fmt.Errorf("%s: %s", e.Code, e.Detail)
		} else {
			// wait for refresh, irrespective of a previous error
			err = api.ErrMustRetry
		}
	}

	return res, err
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()

	if err == nil {
		return float64(res.Attributes.BatteryLevel), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if err == nil {
		if res.Attributes.PlugStatus > 0 {
			status = api.StatusB
		}
		if res.Attributes.ChargeStatus > 1.0 {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()

	if err == nil {
		if res.Attributes.RangeHvacOff == nil {
			return 0, api.ErrNotAvailable
		}

		return int64(*res.Attributes.RangeHvacOff), nil
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()

	if err == nil {
		if res.Attributes.RemainingTime == nil {
			return time.Time{}, api.ErrNotAvailable
		}

		return res.Attributes.LastUpdateTime.Time.Add(time.Duration(*res.Attributes.RemainingTime) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleChargeController = (*Provider)(nil)

// StartCharge implements the api.VehicleChargeController interface
func (v *Provider) StartCharge() error {
	return v.action(ActionChargeStart)
}

// StopCharge implements the api.VehicleChargeController interface
func (v *Provider) StopCharge() error {
	return v.action(ActionChargeStop)
}
