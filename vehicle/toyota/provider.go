package toyota

import (
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const (
	refreshInterval = 15 * time.Minute
	pollInterval    = 5 * time.Minute
)

type Provider struct {
	log         *util.Logger
	status      func() (Status, error)
	refresh     func() error
	lastRefresh time.Time
}

func NewProvider(log *util.Logger, api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{
		log: log,
		refresh: func() error {
			return api.RefreshStatus(vin)
		},
	}

	// Poll at most every 5 min to pick up cloud updates promptly after a
	// realtime-status refresh. The refresh POST wakes the car's TCU which
	// pushes fresh data to the cloud after ~5 min. Returning the current
	// (possibly stale) GET response is intentional — re-reading immediately
	// after the POST would still return stale data.
	impl.status = util.Cached(func() (Status, error) {
		res, err := api.Status(vin)
		if err == nil && strings.EqualFold(res.Payload.ChargingStatus, "charging") && time.Since(impl.lastRefresh) >= refreshInterval {
			impl.lastRefresh = time.Now()
			if err := impl.refresh(); err != nil {
				log.ERROR.Printf("status refresh: %v", err)
			}
		}
		return res, err
	}, min(cache, pollInterval))

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
