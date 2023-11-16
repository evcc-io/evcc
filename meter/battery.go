package meter

import (
	"github.com/evcc-io/evcc/api"
)

type battery struct {
	MinSoc, MaxSoc float64
}

// Decorator returns an api.BatteryController decorator
func (m *battery) BatteryController(socG func() (float64, error), limitSocS func(float64) error) func(api.BatteryMode) error {
	return func(mode api.BatteryMode) error {
		switch mode {
		case api.BatteryNormal:
			return limitSocS(m.MinSoc)

		case api.BatteryHold:
			soc, err := socG()
			if err != nil {
				return err
			}
			return limitSocS(soc)

		case api.BatteryCharge:
			return limitSocS(m.MaxSoc)

		default:
			return api.ErrNotAvailable
		}
	}
}
