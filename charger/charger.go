package charger

import (
	"context"
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/provider"
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

//go:generate go run ../cmd/tools/decorate.go -f decorateCustom -b *Charger -r api.Charger -t "api.ChargerEx,MaxCurrentMillis,func(float64) error" -t "api.Identifier,Identify,func() (string, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.Resurrector,WakeUp,func() error" -t "api.Battery,Soc,func() (float64, error)" -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)"

// NewConfigurableFromConfig creates a new configurable charger
func NewConfigurableFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		embed                               `mapstructure:",squash"`
		Status, Enable, Enabled, MaxCurrent provider.Config
		MaxCurrentMillis                    *provider.Config
		Identify, Phases1p3p                *provider.Config
		Wakeup                              *provider.Config
		Soc                                 *provider.Config
		Tos                                 bool

		// optional measurements
		Power  *provider.Config
		Energy *provider.Config

		Currents, Voltages []provider.Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	status, err := provider.NewStringGetterFromConfig(ctx, cc.Status)
	if err != nil {
		return nil, fmt.Errorf("status: %w", err)
	}

	enabled, err := provider.NewBoolGetterFromConfig(ctx, cc.Enabled)
	if err != nil {
		return nil, fmt.Errorf("enabled: %w", err)
	}

	enable, err := provider.NewBoolSetterFromConfig(ctx, "enable", cc.Enable)
	if err != nil {
		return nil, fmt.Errorf("enable: %w", err)
	}

	maxcurrent, err := provider.NewIntSetterFromConfig(ctx, "maxcurrent", cc.MaxCurrent)
	if err != nil {
		return nil, fmt.Errorf("maxcurrent: %w", err)
	}

	c, err := NewConfigurable(status, enabled, enable, maxcurrent)
	if err != nil {
		return nil, err
	}

	c.embed = &cc.embed

	var maxcurrentmillis func(float64) error
	if cc.MaxCurrentMillis != nil {
		maxcurrentmillis, err = provider.NewFloatSetterFromConfig(ctx, "maxcurrentmillis", *cc.MaxCurrentMillis)
		if err != nil {
			return nil, fmt.Errorf("maxcurrentmillis: %w", err)
		}
	}

	// decorate phases
	var phases1p3p func(int) error
	if cc.Phases1p3p != nil {
		if !cc.Tos {
			return nil, errors.New("1p3p does no longer handle disable/enable. Use tos: true to confirm you understand the consequences")
		}

		phases1p3pS, err := provider.NewIntSetterFromConfig(ctx, "phases", *cc.Phases1p3p)
		if err != nil {
			return nil, fmt.Errorf("phases: %w", err)
		}

		phases1p3p = func(phases int) error {
			return phases1p3pS(int64(phases))
		}
	}

	// decorate identifier
	var identify func() (string, error)
	if cc.Identify != nil {
		identify, err = provider.NewStringGetterFromConfig(ctx, *cc.Identify)
		if err != nil {
			return nil, fmt.Errorf("identify: %w", err)
		}
	}

	// decorate wakeup
	var wakeup func() error
	if cc.Wakeup != nil {
		wakeupS, err := provider.NewBoolSetterFromConfig(ctx, "wakeup", *cc.Wakeup)
		if err != nil {
			return nil, fmt.Errorf("wakeup: %w", err)
		}

		wakeup = func() error {
			return wakeupS(true)
		}
	}

	// decorate soc
	var soc func() (float64, error)
	if cc.Soc != nil {
		soc, err = provider.NewFloatGetterFromConfig(ctx, *cc.Soc)
		if err != nil {
			return nil, fmt.Errorf("soc: %w", err)
		}
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
