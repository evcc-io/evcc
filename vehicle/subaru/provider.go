package subaru

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Provider struct {
	statusG func() (Status, error)
}

func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: util.Cached(func() (Status, error) {
			return api.Status(vin)
		}, cache),
	}
	return impl
}

// status retrieves the vehicle status and validates the response.
// The Subaru API may return invalid data on transient failures: either
// an empty payload (missing range unit) or zeroed-out values with a
// valid unit. In both cases, we signal a retry so evcc keeps the last
// known good values instead of acting on invalid data (e.g. 0% SoC
// triggering unwanted minSoC charging).
func (v *Provider) status() (Status, error) {
	res, err := v.statusG()
	if err != nil {
		return res, err
	}
	if res.Payload.EvRangeWithAc.Unit == "" ||
		res.Payload.BatteryLevel == 0 && res.Payload.EvRangeWithAc.Value == 0 {
		return res, api.ErrMustRetry
	}
	return res, nil
}

func (v *Provider) Soc() (float64, error) {
	res, err := v.status()
	return float64(res.Payload.BatteryLevel), err
}

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.status()
	if err != nil {
		return 0, err
	}
	return res.Payload.EvRangeWithAc.ValueInKilometers()
}
