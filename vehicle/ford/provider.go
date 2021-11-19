package ford

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

const refreshTimeout = 2 * time.Minute

type Provider struct {
	statusG func() (interface{}, error)
	// action      func(action, cmd string) (ActionResponse, error)
	expiry      time.Duration
	refreshTime time.Time
}

func NewProvider(api *API, vin string, expiry, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: provider.NewCached(func() (interface{}, error) {
			return api.Status(vin)
		}, cache).InterfaceGetter(),
		// action: func(action, cmd string) (ActionResponse, error) {
		// 	return api.Action(vin, pin, action, cmd)
		// },
		expiry: expiry,
	}

	return impl
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Battery interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		return res.VehicleStatus.BatteryFillLevel.Value, nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		return int64(res.VehicleStatus.ElVehDTE.Value), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if res, ok := res.(StatusResponse); err == nil && ok {
		if res.VehicleStatus.PlugStatus.Value == 1 {
			status = api.StatusB // connected, not charging
		}
		if res.VehicleStatus.ChargingStatus.Value == "ChargingAC" {
			status = api.StatusC // charging
		}
	}

	return status, err
}
