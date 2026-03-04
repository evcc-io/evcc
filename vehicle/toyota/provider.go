package toyota

import (
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const refreshInterval = 15 * time.Minute

type Provider struct {
	status      func() (Status, error)
	refresh     func() error
	lastRefresh time.Time
}

func NewProvider(log *util.Logger, api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		refresh: func() error {
			return api.RefreshStatus(vin)
		},
	}

	impl.status = util.Cached(func() (Status, error) {
		res, err := api.Status(vin)
		if err == nil && strings.EqualFold(res.Payload.ChargingStatus, "charging") && time.Since(impl.lastRefresh) >= refreshInterval {
			impl.lastRefresh = time.Now()
			if err := impl.refresh(); err != nil {
				log.ERROR.Printf("status refresh: %v", err)
			}
		}
		return res, err
	}, cache)

	return impl
}

var _ api.Resurrector = (*Provider)(nil)

func (v *Provider) WakeUp() error {
	err := v.refresh()
	if err == nil {
		v.lastRefresh = time.Now()
	}
	return err
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
