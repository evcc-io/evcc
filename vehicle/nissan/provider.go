package nissan

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

// Provider is a kamereon provider
type Provider struct {
	statusG     func() (interface{}, error)
	action      func(value Action) error
	expiry      time.Duration
	refreshID   string
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

	impl.statusG = provider.NewCached(func() (interface{}, error) {
		return impl.status(
			func() (Response, error) { return api.BatteryStatus(vin) },
			func() (Response, error) { return api.RefreshRequest(vin, "RefreshBatteryStatus") },
		)
	}, cache).InterfaceGetter()

	return impl
}

func (v *Provider) status(battery func() (Response, error), refresh func() (Response, error)) (Response, error) {
	res, err := battery()

	var ts time.Time
	if err == nil {
		ts, err = time.Parse(timeFormat, res.Data.Attributes.LastUpdateTime)

		// return the current value
		if time.Since(ts) <= v.expiry {
			v.refreshID = ""
			return res, err
		}
	}

	// request a refresh, irrespective of a previous error
	if v.refreshID == "" {
		var refreshRes Response
		if refreshRes, err = refresh(); err == nil {
			err = api.ErrMustRetry

			v.refreshID = refreshRes.Data.ID
			v.refreshTime = time.Now()

			if v.refreshID == "" {
				err = errors.New("refresh failed")
			}
		}

		return res, err
	}

	// refresh finally expired
	if time.Since(v.refreshTime) > refreshTimeout {
		v.refreshID = ""
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

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.statusG()

	if res, ok := res.(Response); err == nil && ok {
		return float64(res.Data.Attributes.BatteryLevel), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if res, ok := res.(Response); err == nil && ok {
		if res.Data.Attributes.PlugStatus > 0 {
			status = api.StatusB
		}
		if res.Data.Attributes.ChargingStatus > 1.0 {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()

	if res, ok := res.(Response); err == nil && ok {
		return int64(res.Data.Attributes.RangeHvacOff), nil
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.statusG()

	if res, ok := res.(Response); err == nil && ok {
		timestamp, err := time.Parse(time.RFC3339, res.Data.Attributes.Timestamp)

		if res.Data.Attributes.RemainingTime == nil {
			return time.Time{}, api.ErrNotAvailable
		}

		return timestamp.Add(time.Duration(*res.Data.Attributes.RemainingTime) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleStartCharge = (*Provider)(nil)

// StartCharge implements the api.VehicleStartCharge interface
func (v *Provider) StartCharge() error {
	return v.action(ActionChargeStart)
}

var _ api.VehicleStopCharge = (*Provider)(nil)

// StopCharge implements the api.VehicleStopCharge interface
func (v *Provider) StopCharge() error {
	return v.action(ActionChargeStop)
}
