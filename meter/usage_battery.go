package meter

import "github.com/evcc-io/evcc/api"

type batteryCapacity struct {
	Capacity float64
}

// var _ api.BatteryCapacity = (*batteryCapacity)(nil)

// Decorator returns an api.BatteryCapacity decorator
func (m *batteryCapacity) Decorator() func() float64 {
	if m.Capacity == 0 {
		return nil
	}
	return func() float64 {
		return m.Capacity
	}
}

type batteryPowerLimits struct {
	MaxChargePower    float64
	MaxDischargePower float64
}

// var _ api.BatteryPowerLimiter = (*batteryPowerLimits)(nil)

// Decorator returns an api.BatteryPowerLimiter decorator
func (m *batteryPowerLimits) Decorator() func() (float64, float64) {
	if m.MaxChargePower == 0 || m.MaxDischargePower == 0 {
		return nil
	}
	return func() (float64, float64) {
		return m.MaxChargePower, m.MaxDischargePower
	}
}

type batterySocLimits struct {
	MinSoc, MaxSoc float64
}

// Decorator returns an api.BatterySocLimiter decorator
func (m *batterySocLimits) Decorator() func() (float64, float64) {
	if m.MinSoc == 0 && m.MaxSoc == 0 {
		return nil
	}
	return func() (float64, float64) {
		return m.MinSoc, m.MaxSoc
	}
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
