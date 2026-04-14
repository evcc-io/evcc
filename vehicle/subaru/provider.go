package subaru

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Provider struct {
	statusG func() (Status, error)
}

func NewProvider(a *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		statusG: util.Cached(func() (Status, error) {
			res, err := a.Status(vin)
			if err != nil {
				return res, err
			}
			if res.Payload.EvRangeWithAc.Unit == "" ||
				res.Payload.BatteryLevel == 0 && res.Payload.EvRangeWithAc.Value == 0 {
				return res, api.ErrMustRetry
			}
			return res, nil
		}, cache),
	}
	return impl
}

func (v *Provider) Soc() (float64, error) {
	res, err := v.statusG()
	return float64(res.Payload.BatteryLevel), err
}

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.statusG()
	if err != nil {
		return 0, err
	}
	return res.Payload.EvRangeWithAc.ValueInKilometers()
}
