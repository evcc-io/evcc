package charger

import (
	"context"
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

// Charger is an api.Charger implementation with configurable getters and setters.
type Charger struct {
	*embed
	statusG     func() (string, error)
	enabledG    func() (bool, error)
	enableS     func(bool) error
	maxCurrentS func(int64) error
}

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

//go:generate decorate -f decorateCustom -b *Charger -r api.Charger -t "api.ChargerEx,MaxCurrentMillis,func(float64) error" -t "api.Identifier,Identify,func() (string, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.Resurrector,WakeUp,func() error" -t "api.Battery,Soc,func() (float64, error)" -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)"

// NewConfigurableFromConfig creates a new configurable charger
func NewConfigurableFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		embed                               `mapstructure:",squash"`
		Status, Enable, Enabled, MaxCurrent plugin.Config
		MaxCurrentMillis                    *plugin.Config
		Identify, Phases1p3p                *plugin.Config
		Wakeup                              *plugin.Config
		Soc                                 *plugin.Config
		Tos                                 bool

		// optional measurements
		Power  *plugin.Config
		Energy *plugin.Config

		Currents, Voltages []plugin.Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	status, err := cc.Status.StringGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("status: %w", err)
	}

	enabled, err := cc.Enabled.BoolGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("enabled: %w", err)
	}

	enable, err := cc.Enable.BoolSetter(ctx, "enable")
	if err != nil {
		return nil, fmt.Errorf("enable: %w", err)
	}

	maxcurrent, err := cc.MaxCurrent.IntSetter(ctx, "maxcurrent")
	if err != nil {
		return nil, fmt.Errorf("maxcurrent: %w", err)
	}

	c, err := NewConfigurable(status, enabled, enable, maxcurrent)
	if err != nil {
		return nil, err
	}

	c.embed = &cc.embed

	maxcurrentmillis, err := cc.MaxCurrentMillis.FloatSetter(ctx, "maxcurrentmillis")
	if err != nil {
		return nil, fmt.Errorf("maxcurrentmillis: %w", err)
	}

	// decorate phases
	var phases1p3p func(int) error
	if cc.Phases1p3p != nil {
		if !cc.Tos {
			return nil, errors.New("1p3p does no longer handle disable/enable. Use tos: true to confirm you understand the consequences")
		}

		phases1p3pS, err := cc.Phases1p3p.IntSetter(ctx, "phases")
		if err != nil {
			return nil, fmt.Errorf("phases: %w", err)
		}

		phases1p3p = func(phases int) error {
			return phases1p3pS(int64(phases))
		}
	}

	// decorate identifier
	identify, err := cc.Identify.StringGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("identify: %w", err)
	}

	// decorate wakeup
	var wakeup func() error
	if cc.Wakeup != nil {
		set, err := cc.Wakeup.BoolSetter(ctx, "wakeup")
		if err != nil {
			return nil, fmt.Errorf("wakeup: %w", err)
		}

		wakeup = func() error {
			return set(true)
		}
	}

	// decorate soc
	soc, err := cc.Soc.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("soc: %w", err)
	}

	// decorate measurements
	powerG, energyG, err := meter.BuildMeasurements(ctx, cc.Power, cc.Energy)
	if err != nil {
		return nil, err
	}

	currentsG, voltagesG, _, err := meter.BuildPhaseMeasurements(ctx, cc.Currents, cc.Voltages, nil)
	if err != nil {
		return nil, err
	}

	return decorateCustom(c, maxcurrentmillis, identify, phases1p3p, wakeup, soc, powerG, energyG, currentsG, voltagesG), nil
}

// NewConfigurable creates a new charger
func NewConfigurable(
	statusG func() (string, error),
	enabledG func() (bool, error),
	enableS func(bool) error,
	maxCurrentS func(int64) error,
) (*Charger, error) {
	c := &Charger{
		embed:       new(embed),
		statusG:     statusG,
		enabledG:    enabledG,
		enableS:     enableS,
		maxCurrentS: maxCurrentS,
	}

	return c, nil
}

// Status implements the api.Charger interface
func (m *Charger) Status() (api.ChargeStatus, error) {
	s, err := m.statusG()
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatusString(s)
}

// Enabled implements the api.Charger interface
func (m *Charger) Enabled() (bool, error) {
	return m.enabledG()
}

// Enable implements the api.Charger interface
func (m *Charger) Enable(enable bool) error {
	return m.enableS(enable)
}

// MaxCurrent implements the api.Charger interface
func (m *Charger) MaxCurrent(current int64) error {
	return m.maxCurrentS(current)
}
