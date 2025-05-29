package meter

import "github.com/evcc-io/evcc/api"

type batteryCapacity struct {
	Capacity float64
}

// var _ api.BatteryCapacity = (*batteryCapacity)(nil)

// Decorator returns the capacity decorator
func (m *batteryCapacity) Decorator() func() float64 {
	if m.Capacity == 0 {
		return nil
	}
	return func() float64 {
		return m.Capacity
	}
}

type batteryMaxACPower struct {
	MaxACPower float64
}

// var _ api.MaxACPowerGetter = (*batteryMaxACPower)(nil)

// Decorator returns the max AC power decorator
func (m *batteryMaxACPower) Decorator() func() float64 {
	if m.MaxACPower == 0 {
		return nil
	}
	return func() float64 {
		return m.MaxACPower
	}
}

type batterySocLimits struct {
	MinSoc, MaxSoc float64
}

// LimitController returns an api.BatteryController decorator
func (m *batterySocLimits) LimitController(socG func() (float64, error), limitSocS func(float64) error) func(api.BatteryMode) error {
	return func(mode api.BatteryMode) error {
		switch mode {
		case api.BatteryNormal:
			return limitSocS(m.MinSoc)

		case api.BatteryHold:
			soc, err := socG()
			if err != nil {
				return err
			}
			return limitSocS(min(100, max(soc, m.MinSoc)))

		case api.BatteryCharge:
			return limitSocS(m.MaxSoc)

		default:
			return api.ErrNotAvailable
		}
	}
}
