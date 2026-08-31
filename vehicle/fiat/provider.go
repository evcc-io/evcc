package fiat

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const refreshTimeout = 2 * time.Minute

type Provider struct {
	statusG     func() (StatusResponse, error)
	locationG   func() (LocationResponse, error)
	action      func(action, cmd string) (ActionResponse, error)
	expiry      time.Duration
	refreshTime time.Time
}

func NewProvider(api *API, vin, pin string, expiry, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: util.Cached(func() (StatusResponse, error) {
			return api.Status(vin)
		}, cache),
		locationG: util.Cached(func() (LocationResponse, error) {
			return api.Location(vin)
		}, cache),
		action: func(action, cmd string) (ActionResponse, error) {
			return api.Action(vin, pin, action, cmd)
		},
		expiry: expiry,
	}

	// use pin for refreshing
	if pin != "" {
		impl.statusG = util.Cached(func() (StatusResponse, error) {
			return impl.status(
				func() (StatusResponse, error) { return api.Status(vin) },
			)
		}, cache)
	}

	return impl
}

// deepRefresh triggers a deep refresh of the vehicle data. It returns true if the
// refresh was accepted (pending). A vehicle that is not plugged in refuses the
// refresh with 403 Forbidden; in that case retrying is pointless, so it returns
// false without error. A 403 from the preceding pin authentication is a real
// error and is surfaced instead of being masked.
func (v *Provider) deepRefresh() (bool, error) {
	res, err := v.action("ev", "DEEPREFRESH")
	if err != nil {
		if se, ok := errors.AsType[*request.StatusError](err); ok && se.StatusCode() == http.StatusForbidden {
			if req := se.Response().Request; req == nil || !strings.HasSuffix(req.URL.Path, "/authenticate") {
				return false, nil
			}
		}
		return false, err
	}
	if res.ResponseStatus != "pending" {
		return false, fmt.Errorf("invalid response status: %s", res.ResponseStatus)
	}
	return true, nil
}

func (v *Provider) status(statusG func() (StatusResponse, error)) (StatusResponse, error) {
	res, err := statusG()

	// handle refresh
	if err == nil {
		// result expired?
		if res.Timestamp.Add(v.expiry).Before(time.Now()) {
			// start refresh
			if v.refreshTime.IsZero() {
				accepted, err := v.deepRefresh()
				if err != nil {
					return res, err
				}

				// vehicle refused the refresh (e.g. not plugged in); retrying is
				// pointless, so return the last known data instead of blocking
				if !accepted {
					return res, nil
				}

				v.refreshTime = time.Now()
				return res, api.ErrMustRetry
			}

			// wait for refresh
			if time.Since(v.refreshTime) > refreshTimeout {
				v.refreshTime = time.Time{}
				return res, api.ErrTimeout
			}

			return res, api.ErrMustRetry
		}

		// refresh done
		v.refreshTime = time.Time{}
	}

	return res, err
}

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	if err == nil {
		if res.EvInfo == nil {
			return 0, api.ErrNotAvailable
		}

		return res.EvInfo.Battery.StateOfCharge, nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err == nil {
		if res.EvInfo == nil {
			return 0, api.ErrNotAvailable
		}

		return int64(res.EvInfo.Battery.DistanceToEmpty.Value), nil
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	if err == nil {
		return float64(res.VehicleInfo.Odometer.Odometer.Value), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if err == nil {
		if res.EvInfo == nil {
			return api.StatusNone, api.ErrNotAvailable
		}

		if res.EvInfo.Battery.PlugInStatus {
			status = api.StatusB // connected, not charging
		}
		if res.EvInfo.Battery.ChargingStatus == "CHARGING" {
			status = api.StatusC // charging
		}
	}

	return status, err
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements the api.VehiclePosition interface
func (v *Provider) Position() (float64, float64, error) {
	res, err := v.locationG()
	if err == nil {
		return res.Latitude, res.Longitude, nil
	}

	return 0, 0, err
}
