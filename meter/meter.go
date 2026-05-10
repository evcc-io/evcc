package meter

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/meter/measurement"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates a new meter from config
func NewConfigurableFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		measurement.Energy `mapstructure:",squash"` // energy optional
		measurement.Phases `mapstructure:",squash"` // optional

		// pv
		pvMaxACPower `mapstructure:",squash"`

		// battery
		batteryCapacity    `mapstructure:",squash"`
		batterySocLimits   `mapstructure:",squash"`
		batteryPowerLimits `mapstructure:",squash"`
		Soc                *plugin.Config // optional
		LimitSoc           *plugin.Config // optional
		BatteryMode        *plugin.Config // optional
	}{
		batterySocLimits: batterySocLimits{
			MinSoc: 20,
			MaxSoc: 95,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	powerG, importG, exportG, err := cc.Energy.Configure(ctx)
	if err != nil {
		return nil, err
	}

	m, _ := NewConfigurable(powerG)
	implement.May(m, implement.MeterImport(importG))
	implement.May(m, implement.MeterExport(exportG))

	// decorate soc
	socG, err := cc.Soc.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("battery soc: %w", err)
	}

	if socG != nil {
		implement.Has(m, implement.Battery(socG))
		implement.May(m, implement.BatteryCapacity(cc.batteryCapacity.Decorator()))
		implement.May(m, implement.BatterySocLimiter(cc.batterySocLimits.Decorator()))
		implement.May(m, implement.BatteryPowerLimiter(cc.batteryPowerLimits.Decorator()))

		switch {
		case cc.Soc != nil && cc.LimitSoc != nil:
			limitSocS, err := cc.LimitSoc.FloatSetter(ctx, "limitSoc")
			if err != nil {
				return nil, fmt.Errorf("battery limit soc: %w", err)
			}

			implement.Has(m, implement.BatteryController(cc.batterySocLimits.LimitController(socG, limitSocS)))

		case cc.BatteryMode != nil:
			modeS, err := cc.BatteryMode.IntSetter(ctx, "batteryMode")
			if err != nil {
				return nil, fmt.Errorf("battery mode: %w", err)
			}

			implement.Has(m, implement.BatteryController(func(mode api.BatteryMode) error {
				return modeS(int64(mode))
			}))
		}

		return m, nil
	}

	currentsG, voltagesG, powersG, err := cc.Phases.Configure(ctx)
	if err != nil {
		return nil, err
	}

	implement.May(m, implement.PhaseCurrents(currentsG))
	implement.May(m, implement.PhaseVoltages(voltagesG))
	implement.May(m, implement.PhasePowers(powersG))
	implement.May(m, implement.MaxACPowerGetter(cc.pvMaxACPower.Decorator()))

	return m, nil
}

// NewConfigurable creates a new meter
func NewConfigurable(currentPowerG func() (float64, error)) (*Meter, error) {
	m := &Meter{
		Caps:          implement.New(),
		currentPowerG: currentPowerG,
	}
	return m, nil
}

// Meter is an api.Meter implementation with configurable getters and setters.
type Meter struct {
	implement.Caps
	currentPowerG func() (float64, error)
}

// CurrentPower implements the api.Meter interface
func (m *Meter) CurrentPower() (float64, error) {
	return m.currentPowerG()
}
