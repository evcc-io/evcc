package meter

import (
	"github.com/evcc-io/evcc/api"
)

type battery struct {
	MinSoc, MaxSoc float64
}

// LimitController returns an api.BatteryController decorator
func (m *battery) LimitController(socG func() (float64, error), limitSocS func(float64) error) func(api.BatteryMode) error {
	return func(mode api.BatteryMode) error {
		switch mode {
		case api.BatteryNormal:
			return limitSocS(m.MinSoc)

		case api.BatteryHold:
			soc, err := socG()
			if err != nil {
				return err
			}
			return limitSocS(max(soc, m.MinSoc))

		case api.BatteryCharge:
			return limitSocS(m.MaxSoc)

		default:
			return api.ErrNotAvailable
		}
	}
}

// ModeController returns an api.BatteryController decorator
func (m *battery) ModeController(modeS func(int64) error) func(api.BatteryMode) error {
	return func(mode api.BatteryMode) error {
		return modeS(int64(mode))
	}
}
