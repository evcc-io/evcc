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
	status  func() (Status, error)
	refresh func() error
}

func NewProvider(log *util.Logger, api *API, vin string, cache time.Duration) *Provider {
	var (
		mu          sync.Mutex
		lastRefresh time.Time
	)

	refresh := func() error {
		err := api.RefreshStatus(vin)
		if err == nil {
			mu.Lock()
			lastRefresh = time.Now()
			mu.Unlock()
		}
		return err
	}

	impl := &Provider{
		status: util.Cached(func() (Status, error) {
			res, err := api.Status(vin)
			if err != nil {
				return res, err
			}

			// While charging, periodically ask the TCU to push fresh data
			// to the cloud so subsequent polls return up-to-date SOC values.
			mu.Lock()
			needsRefresh := strings.EqualFold(res.Payload.ChargingStatus, "charging") && time.Since(lastRefresh) >= refreshInterval
			mu.Unlock()
			if needsRefresh {
				if err := refresh(); err != nil {
					log.ERROR.Printf("status refresh: %v", err)
				}
			}

			return res, nil
		}, cache),
		refresh: refresh,
	}

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
