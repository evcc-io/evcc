package bluelink

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
)

// Provider implements the Kia/Hyundai bluelink api.
// Based on https://github.com/Hacksore/bluelinky.
type Provider struct {
	apiG func() (interface{}, error)
}

// New creates a new BlueLink API
func NewProvider(api *API, vid string, cache time.Duration) *Provider {
	v := &Provider{
		apiG: provider.NewCached(func() (interface{}, error) {
			return api.Status(vid)
		}, cache).InterfaceGetter(),
	}

	return v
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Battery interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.apiG()

	if res, ok := res.(StatusData); err == nil && ok {
		return float64(res.EvStatus.BatteryStatus), nil
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.apiG()

	if res, ok := res.(StatusData); err == nil && ok {
		remaining := res.EvStatus.RemainTime2.Atc.Value

		if remaining == 0 {
			return time.Time{}, api.ErrNotAvailable
		}

		ts, err := res.Updated()
		return ts.Add(time.Duration(remaining) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.apiG()

	if res, ok := res.(StatusData); err == nil && ok {
		if dist := res.EvStatus.DrvDistance; len(dist) == 1 {
			return int64(dist[0].RangeByFuel.EvModeRange.Value), nil
		}

		return 0, api.ErrNotAvailable
	}

	return 0, err
}
