package meter

import (
	"github.com/evcc-io/evcc/api"
)

type battery struct {
	MinSoc            float64
	MaxDischargePower float64
}

// var _ api.BatteryController = (*battery)(nil)

// Decorator returns a battery controller decorator
func (m *battery) BatteryController(dischargePowerS, minSocS func(float64) error) (func() (api.BatteryMode, error), func(api.BatteryMode) error) {
	get := func() (api.BatteryMode, error) {
		return api.BatteryNormal, nil
	}

	set := func(mode api.BatteryMode) error {
		switch mode {
		case api.BatteryNormal:
			// normal discharge
			if err := dischargePowerS(m.MaxDischargePower); err != nil {
				return err
			}
			// default minsoc
			return minSocS(m.MinSoc)

		case api.BatteryLocked:
			// lock discharge
			if err := dischargePowerS(0); err != nil {
				return err
			}
			// default minsoc
			return minSocS(m.MinSoc)

		case api.BatteryCharge:
			// normal discharge
			if err := dischargePowerS(m.MaxDischargePower); err != nil {
				return err
			}
			// full charge
			return minSocS(95)

		default:
			return api.ErrNotAvailable
		}
	}

	return get, set
}
