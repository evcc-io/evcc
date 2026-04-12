package meter

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/measurement"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

//go:generate go tool decorate

//evcc:function decorateMeter
//evcc:basetype api.Meter
//evcc:types api.MeterEnergy,api.PhaseCurrents,api.PhaseVoltages,api.PhasePowers,api.MaxACPowerGetter

//evcc:function decorateMeterBattery
//evcc:basetype api.Meter
//evcc:types api.MeterEnergy,api.Battery,api.BatteryCapacity,api.BatterySocLimiter,api.BatteryPowerLimiter,api.BatteryController

// NewConfigurableFromConfig creates api.Meter from config
func NewConfigurableFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		measurement.Energy `mapstructure:",squash"` // energy optional
		measurement.Phases `mapstructure:",squash"` // optional

		// pv
		pvMaxACPower `mapstructure:",squash"`

		// pv curtailment
		Curtail      *plugin.Config // optional: float setter, CurtailLimit=curtailed, FeedInLimit=full power
		Curtailed    *plugin.Config // optional: float getter, <FeedInLimit means curtailed
		CurtailLimit float64        // optional: power limit in % when curtailed (default: 0)
		FeedInLimit  float64        // optional: power limit in % when not curtailed (default: 100)

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
		FeedInLimit: 100,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	powerG, energyG, err := cc.Energy.Configure(ctx)
	if err != nil {
		return nil, err
	}

	currentsG, voltagesG, powersG, err := cc.Phases.Configure(ctx)
	if err != nil {
		return nil, err
	}

	m, _ := NewConfigurable(powerG)

	// decorate soc
	socG, err := cc.Soc.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("battery soc: %w", err)
	}

	var batModeS func(api.BatteryMode) error

	switch {
	case cc.Soc != nil && cc.LimitSoc != nil:
		limitSocS, err := cc.LimitSoc.FloatSetter(ctx, "limitSoc")
		if err != nil {
			return nil, fmt.Errorf("battery limit soc: %w", err)
		}

		batModeS = cc.batterySocLimits.LimitController(socG, limitSocS)

	case cc.BatteryMode != nil:
		modeS, err := cc.BatteryMode.IntSetter(ctx, "batteryMode")
		if err != nil {
			return nil, fmt.Errorf("battery mode: %w", err)
		}

		batModeS = func(mode api.BatteryMode) error {
			return modeS(int64(mode))
		}
	}

	if socG != nil {
		return m.DecorateBattery(
			energyG,
			socG, cc.batteryCapacity.Decorator(),
			cc.batterySocLimits.Decorator(), cc.batteryPowerLimits.Decorator(),
			batModeS,
		), nil
	}

	base := m.Decorate(
		energyG, currentsG, voltagesG, powersG, cc.pvMaxACPower.Decorator(),
	)

	// optionally wrap with curtailment support
	if cc.Curtail != nil || cc.Curtailed != nil {
		var curtailS func(float64) error
		if cc.Curtail != nil {
			curtailS, err = cc.Curtail.FloatSetter(ctx, "curtail")
			if err != nil {
				return nil, fmt.Errorf("curtail: %w", err)
			}
		}

		var curtailedG func() (float64, error)
		if cc.Curtailed != nil {
			curtailedG, err = cc.Curtailed.FloatGetter(ctx)
			if err != nil {
				return nil, fmt.Errorf("curtailed: %w", err)
			}
		}

		return &curtailMeter{
			Meter:        base,
			curtailS:     curtailS,
			curtailedG:   curtailedG,
			curtailLimit: cc.CurtailLimit,
			nominalLimit: cc.FeedInLimit,
		}, nil
	}

	return base, nil
}

// NewConfigurable creates a new meter
func NewConfigurable(currentPowerG func() (float64, error)) (*Meter, error) {
	m := &Meter{
		currentPowerG: currentPowerG,
	}
	return m, nil
}

// Meter is an api.Meter implementation with configurable getters and setters.
type Meter struct {
	currentPowerG func() (float64, error)
}

// Decorate attaches additional capabilities to the base meter
func (m *Meter) Decorate(
	totalEnergy func() (float64, error),
	currents, voltages, powers func() (float64, float64, float64, error),
	maxACPower func() float64,
) api.Meter {
	return decorateMeter(m,
		totalEnergy, currents, voltages, powers,
		maxACPower,
	)
}

func (m *Meter) DecorateBattery(
	totalEnergy func() (float64, error),
	soc func() (float64, error), capacity func() float64,
	socLimits, powerLimits func() (float64, float64),
	setMode func(api.BatteryMode) error,
) api.Meter {
	return decorateMeterBattery(m,
		totalEnergy,
		soc, capacity,
		socLimits, powerLimits,
		setMode,
	)
}

// CurrentPower implements the api.Meter interface
func (m *Meter) CurrentPower() (float64, error) {
	return m.currentPowerG()
}

// curtailMeter wraps api.Meter with api.Curtailer support.
// curtailS is called with curtailLimit to enable curtailment and nominalLimit to disable it.
// curtailedG returns the current active power limit (0-100); values below nominalLimit indicate curtailment.
type curtailMeter struct {
	api.Meter
	curtailS     func(float64) error
	curtailedG   func() (float64, error)
	curtailLimit float64 // power limit in % written when curtailing (default: 0)
	nominalLimit float64 // power limit in % written when not curtailing (default: 100)
}

var _ api.Curtailer = (*curtailMeter)(nil)

// Curtail implements api.Curtailer.
// Curtail(true) writes curtailLimit (default 0 %, PV output suppressed).
// Curtail(false) writes nominalLimit (default 100 %, full or legally-capped PV output).
func (m *curtailMeter) Curtail(curtail bool) error {
	if m.curtailS == nil {
		return api.ErrNotAvailable
	}
	val := m.nominalLimit
	if curtail {
		val = m.curtailLimit
	}
	return m.curtailS(val)
}

// Curtailed implements api.Curtailer.
// Returns true when the active power limit is below nominalLimit.
func (m *curtailMeter) Curtailed() (bool, error) {
	if m.curtailedG == nil {
		return false, api.ErrNotAvailable
	}
	val, err := m.curtailedG()
	return val < m.nominalLimit, err
}
