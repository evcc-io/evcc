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
	return resolveFloat(ctx, m.Capacity)
}

// resolveFloat resolves a static number or float plugin config to a getter.
// nil/zero static returns a nil getter (not configured).
func resolveFloat(ctx context.Context, v any) (func() float64, error) {
	switch v := v.(type) {
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
				return 0 // ponytail: treat plugin error as unknown value
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

// floatOr0 evaluates g, returning 0 for a nil (unconfigured) getter.
func floatOr0(g func() float64) float64 {
	if g == nil {
		return 0
	}
	return g()
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

type batteryPowerLimitsCtx struct {
	MaxChargePower    any // static W value or float plugin
	MaxDischargePower any // static W value or float plugin
}

// var _ api.BatteryPowerLimiter = (*batteryPowerLimitsCtx)(nil)

// Decorator returns an api.BatteryPowerLimiter decorator. Each limit may be a
// static number or a float plugin config; either unset means not configured.
func (m *batteryPowerLimitsCtx) Decorator(ctx context.Context) (func() (float64, float64), error) {
	charge, err := resolveFloat(ctx, m.MaxChargePower)
	if err != nil {
		return nil, err
	}
	discharge, err := resolveFloat(ctx, m.MaxDischargePower)
	if err != nil {
		return nil, err
	}
	if charge == nil || discharge == nil {
		return nil, nil
	}
	return func() (float64, float64) {
		return charge(), discharge()
	}, nil
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

type batterySocLimitsCtx struct {
	MinSoc, MaxSoc any // static % value or float plugin
}

// var _ api.BatterySocLimiter = (*batterySocLimitsCtx)(nil)

func (m *batterySocLimitsCtx) getters(ctx context.Context) (func() float64, func() float64, error) {
	minG, err := resolveFloat(ctx, m.MinSoc)
	if err != nil {
		return nil, nil, err
	}
	maxG, err := resolveFloat(ctx, m.MaxSoc)
	if err != nil {
		return nil, nil, err
	}
	return minG, maxG, nil
}

// Decorator returns an api.BatterySocLimiter decorator. Each limit may be a
// static number or a float plugin config; both unset means not configured.
func (m *batterySocLimitsCtx) Decorator(ctx context.Context) (func() (float64, float64), error) {
	minG, maxG, err := m.getters(ctx)
	if err != nil {
		return nil, err
	}
	if minG == nil && maxG == nil {
		return nil, nil
	}
	return func() (float64, float64) {
		return floatOr0(minG), floatOr0(maxG)
	}, nil
}

// LimitController returns an api.BatteryController decorator
func (m *batterySocLimitsCtx) LimitController(ctx context.Context, socG func() (float64, error), limitSocS func(float64) error) (func(api.BatteryMode) error, error) {
	minG, maxG, err := m.getters(ctx)
	if err != nil {
		return nil, err
	}
	return func(mode api.BatteryMode) error {
		switch mode {
		case api.BatteryNormal:
			return limitSocS(floatOr0(minG))

		case api.BatteryHold:
			soc, err := socG()
			if err != nil {
				return err
			}
			return limitSocS(min(100, max(soc, floatOr0(minG))))

		case api.BatteryCharge:
			return limitSocS(floatOr0(maxG))

		// BatteryHoldCharge not implementable via limit soc
		default:
			return api.ErrNotAvailable
		}
	}, nil
}
