package smart

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

// https://github.com/TA2k/ioBroker.smart-eq

type Provider struct {
	statusG func() (StatusResponse, error)
	expiry  time.Duration
}

func NewProvider(log *util.Logger, api *API, vin string, expiry, cache time.Duration) *Provider {
	v := &Provider{
		expiry: expiry,
	}

	v.statusG = provider.Cached(func() (StatusResponse, error) {
		return v.status(
			func() (StatusResponse, error) { return api.Status(vin) },
			func() (StatusResponse, error) { return api.Refresh(vin) },
		)
	}, cache)

	return v
}

func (v *Provider) status(statusG, refreshG func() (StatusResponse, error)) (StatusResponse, error) {
	res, err := statusG()

	// if ts := res.Status.Data.Soc.Ts.Time; err == nil && ts.Add(v.expiry).Before(time.Now()) {
	// 	fmt.Println("--------------------------", ts)
	// 	res, err = refreshG()
	// 	ts := res.Status.Data.Soc.Ts.Time
	// 	fmt.Println("--------------------------", ts)
	// }

	return res, err
}

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	return res.Status.Data.Soc.Value, err
}

var _ api.ChargeState = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	res, err := v.statusG()
	if err != nil {
		return api.StatusNone, err
	}

	cs := res.PreCond.Data.ChargingStatus
	if cs.Status != 0 {
		return api.StatusNone, fmt.Errorf("unknown status/value: %d/%d", cs.Status, cs.Value)
	}

	// confirmed status/value/active combinations (https://github.com/evcc-io/evcc/discussions/5596#discussioncomment-4556035)
	// 0/0/active: charging
	// 0/2/*:      connected
	// 0/3/*:      disconnected
	switch cs.Value {
	case 0:
		if res.PreCond.Data.ChargingActive.Value {
			return api.StatusC, nil
		}
		return api.StatusB, nil
	case 1, 2:
		return api.StatusB, nil
	case 3:
		return api.StatusA, nil
	default:
		return api.StatusNone, fmt.Errorf("unknown status/value: %d/%d", cs.Status, cs.Value)
	}
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	return int64(res.Status.Data.RangeElectric.Value), err
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the Provider.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	res, err := v.statusG()
	return res.Status.Data.Odo.Value, err
}
