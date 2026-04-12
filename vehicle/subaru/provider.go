package subaru

import (
	"time"

	evccapi "github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Provider struct {
	statusG func() (Status, error)
}

func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		// Validation inside the cached function ensures that ErrMustRetry is
		// stored in the cache's error slot, so the next call forces a re-fetch
		// instead of serving stale bad data for the entire cache TTL.
		statusG: util.Cached(func() (Status, error) {
			res, err := api.Status(vin)
			if err != nil {
				return res, err
			}
			if res.Payload.EvRangeWithAc.Unit == "" ||
				res.Payload.BatteryLevel == 0 && res.Payload.EvRangeWithAc.Value == 0 {
				return res, evccapi.ErrMustRetry
			}
			return res, nil
		}, cache),
	}
	return impl
}

func (v *Provider) status() (Status, error) {
	return v.statusG()
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
