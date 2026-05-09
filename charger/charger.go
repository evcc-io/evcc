package charger

import (
	"context"
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/charger/measurement"
	meter "github.com/evcc-io/evcc/meter/measurement"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

// Charger is an api.Charger implementation with configurable getters and setters.
type Charger struct {
	*embed
	implement.Caps
	statusG     func() (string, error)
	enabledG    func() (bool, error)
	enableS     func(bool) error
	maxCurrentS func(int64) error
}

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates a new charger from config
func NewConfigurableFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	var cc struct {
		embed                               `mapstructure:",squash"`
		Status, Enable, Enabled, MaxCurrent plugin.Config
		MaxCurrentMillis                    *plugin.Config
		Identify, Phases1p3p                *plugin.Config
		Wakeup                              *plugin.Config
		Soc                                 *plugin.Config
		LimitSoc                            *plugin.Config
		FinishTime                          *plugin.Config
		Tos                                 bool
		measurement.Temperature             `mapstructure:",squash"` // optional, for heating devices
		measurement.Energy                  `mapstructure:",squash"` // optional
		meter.Phases                        `mapstructure:",squash"` // optional
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
	c.Caps = implement.New()

	maxcurrentmillis, err := cc.MaxCurrentMillis.FloatSetter(ctx, "maxcurrentmillis")
	if err != nil {
		return nil, fmt.Errorf("maxcurrentmillis: %w", err)
	}
	implement.May(c, implement.ChargerEx(maxcurrentmillis))

	// decorate phases
	if cc.Phases1p3p != nil {
		if !cc.Tos {
			return nil, errors.New("1p3p does no longer handle disable/enable. Use tos: true to confirm you understand the consequences")
		}

		phases1p3pS, err := cc.Phases1p3p.IntSetter(ctx, "phases")
		if err != nil {
			return nil, fmt.Errorf("phases: %w", err)
		}

		implement.Has(c, implement.PhaseSwitcher(func(phases int) error {
			return phases1p3pS(int64(phases))
		}))
	}

	// decorate identifier
	identify, err := cc.Identify.StringGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("identify: %w", err)
	}
	implement.May(c, implement.Identifier(identify))

	// decorate wakeup
	if cc.Wakeup != nil {
		wakeup, err := cc.Wakeup.BoolSetter(ctx, "wakeup")
		if err != nil {
			return nil, fmt.Errorf("wakeup: %w", err)
		}

		implement.Has(c, implement.Resurrector(func() error {
			return wakeup(true)
		}))
	}

	// decorate soc; for heating devices (api.Heating feature), the soc slot holds
	// temperature in °C — fall back to temp getter when no soc getter is configured.
	soc, err := cc.Soc.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("soc: %w", err)
	}

	// decorate limitsoc; similarly, fall back to limittemp getter when no limitsoc is configured.
	limitsoc, err := cc.LimitSoc.IntGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("limitsoc: %w", err)
	}

	// heating fallbacks
	temp, limitTemp, err := cc.Temperature.Configure(ctx)
	if err != nil {
		return nil, err
	}
	if soc == nil && temp != nil {
		soc = temp
	}
	if limitsoc == nil && limitTemp != nil {
		limitsoc = limitTemp
	}
	implement.May(c, implement.Battery(soc))
	implement.May(c, implement.SocLimiter(limitsoc))

	// decorate measurements
	powerG, importG, _, err := cc.Energy.Configure(ctx)
	if err != nil {
		return nil, err
	}
	implement.May(c, implement.Meter(powerG))
	implement.May(c, implement.MeterImport(importG))

	currentsG, voltagesG, _, err := cc.Phases.Configure(ctx)
	if err != nil {
		return nil, err
	}
	implement.May(c, implement.PhaseCurrents(currentsG))
	implement.May(c, implement.PhaseVoltages(voltagesG))

	// decorate finishtime
	finishTime, err := cc.FinishTime.TimeGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("finishTime: %w", err)
	}
	implement.May(c, implement.VehicleFinishTimer(finishTime))

	return c, nil
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
