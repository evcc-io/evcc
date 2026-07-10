package meter

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

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

type batteryCapacityCtx struct {
	Capacity any // static kWh value or float plugin
}

// var _ api.BatteryCapacity = (*batteryCapacityCtx)(nil)

// Decorator returns an api.BatteryCapacity decorator. Capacity may be a static
// number or a float plugin config; nil/zero means not configured.
func (m *batteryCapacityCtx) Decorator(ctx context.Context) (func() float64, error) {
	switch v := m.Capacity.(type) {
	case nil:
		return nil, nil
	case int:
		return staticCapacity(float64(v)), nil
	case int64:
		return staticCapacity(float64(v)), nil
	case float64:
		return staticCapacity(v), nil
	default:
		var cfg plugin.Config
		if err := util.DecodeOther(v, &cfg); err != nil {
			return nil, err
		}
		get, err := cfg.FloatGetter(ctx)
		if err != nil {
			return nil, err
		}
		return func() float64 {
			f, err := get()
			if err != nil {
				return 0 // ponytail: treat plugin error as unknown capacity
			}
			return f
		}, nil
	}
}

func staticCapacity(f float64) func() float64 {
	if f == 0 {
		return nil
	}
	return func() float64 { return f }
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

// var _ api.BatterySocLimiter = (*batterySocLimits)(nil)

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

		// BatteryHoldCharge implementable via limit soc
		default:
			return api.ErrNotAvailable
		}
	}
}
