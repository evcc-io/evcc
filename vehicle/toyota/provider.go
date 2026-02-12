package toyota

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/util"
)

type Provider struct {
	status func() (Status, error)
}

const kmPerMile = 1.609344

func convertToKm(value float64, unit string) (int64, error) {
	// Intentionally truncate fractional kilometers (via int64 conversion)
	// to avoid overreporting the available range.
	switch unit {
	case "km":
		return int64(value), nil
	case "mi":
		return int64(value * kmPerMile), nil
	default:
		return 0, fmt.Errorf("unsupported unit type: %s", unit)
	}
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
	return float64(res.Payload.BatteryLevel), err
}

// Range implements the api.VehicleRange interface.
func (v *Provider) Range() (int64, error) {
	res, err := v.status()
	if err != nil {
		return 0, err
	}

	return convertToKm(res.Payload.EvRangeWithAc.Value, res.Payload.EvRangeWithAc.Unit)
}
