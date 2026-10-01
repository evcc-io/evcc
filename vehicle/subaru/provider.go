package subaru

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const retryTimeout = 2 * time.Minute

type Provider struct {
	status       func() (Status, error)
	incompleteAt time.Time
}

func NewProvider(a *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{}
	impl.status = util.Cached(func() (Status, error) {
		res, err := a.Status(vin)
		if err != nil {
			return res, err
		}
		return res, impl.validate(res)
	}, cache)
	return impl
}

func (v *Provider) validate(res Status) error {
	if !res.Incomplete() {
		v.incompleteAt = time.Time{}
		return nil
	}

	if v.incompleteAt.IsZero() {
		v.incompleteAt = time.Now()
	}

	if time.Since(v.incompleteAt) > retryTimeout {
		return api.ErrTimeout
	}

	return api.ErrMustRetry
}

func (v *Provider) Soc() (float64, error) {
	res, err := v.status()
	return float64(res.Payload.BatteryLevel), err
}

func (v *Provider) Range() (int64, error) {
	res, err := v.status()
	if err != nil {
		return 0, err
	}
	return res.Payload.EvRangeWithAc.ValueInKilometers()
}
