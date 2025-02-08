package toyota

import (
	"time"

	"github.com/evcc-io/evcc/util"
)

type Provider struct {
	status func() (Status, error)
}

func NewProvider(api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		status: util.Cached(func() (Status, error) {
			return api.Status(vin)
		}, cache),
	}
	return impl
}

func (v *Provider) Soc() (float64, error) {
	res, err := v.status()

	if err == nil {
		return float64(res.Payload.BatteryLevel), nil
	}

	return 0, err
}
