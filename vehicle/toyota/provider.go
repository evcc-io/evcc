package toyota

import (
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const refreshInterval = 15 * time.Minute

type Provider struct {
	mu          sync.Mutex
	status      func() (Status, error)
	refresh     func() error
	lastRefresh time.Time
}

func NewProvider(log *util.Logger, api *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{}

	impl.refresh = func() error {
		err := api.RefreshStatus(vin)
		if err == nil {
			impl.mu.Lock()
			impl.lastRefresh = time.Now()
			impl.mu.Unlock()
		}
		return err
	}

	impl.status = util.Cached(func() (Status, error) {
		res, err := api.Status(vin)
		// While charging, periodically ask the TCU to push fresh data
		// to the cloud so subsequent polls return up-to-date SOC values.
		impl.mu.Lock()
		needsRefresh := err == nil && strings.EqualFold(res.Payload.ChargingStatus, "charging") && time.Since(impl.lastRefresh) >= refreshInterval
		impl.mu.Unlock()
		if needsRefresh {
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
	return v.refresh()
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
