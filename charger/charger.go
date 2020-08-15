package charger

import (
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
)

// Charger is an api.Charger implementation with configurable getters and setters.
type Charger struct {
	statusG     func() (string, error)
	enabledG    func() (bool, error)
	enableS     func(bool) error
	maxCurrentS func(int64) error
}

func init() {
	registry.Add("default", NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates a new configurable charger
func NewConfigurableFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Status, Enable, Enabled, MaxCurrent provider.Config
		Phases                              *provider.Config
	}{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	for k, v := range map[string]string{
		"status":     cc.Status.Type,
		"enable":     cc.Enable.Type,
		"enabled":    cc.Enabled.Type,
		"maxcurrent": cc.MaxCurrent.Type,
	} {
		if v == "" {
			return nil, fmt.Errorf("default charger config: %s required", k)
		}
	}

	status, err := provider.NewStringGetterFromConfig(cc.Status)

	var enabled func() (bool, error)
	if err == nil {
		enabled, err = provider.NewBoolGetterFromConfig(cc.Enabled)
	}

	var enable func(bool) error
	if err == nil {
		enable, err = provider.NewBoolSetterFromConfig("enable", cc.Enable)
	}

	var maxcurrent func(int64) error
	if err == nil {
		maxcurrent, err = provider.NewIntSetterFromConfig("maxcurrent", cc.MaxCurrent)
	}

	var c api.Charger
	if err == nil {
		c, err = NewConfigurable(status, enabled, enable, maxcurrent)
	}

	// decorate Charger with ChargePhases
	if err == nil && cc.Phases != nil {
		cp, err := NewChargePhases(*cc.Phases)
		if err != nil {
			return nil, err
		}

		type PhasesDecorator struct {
			api.Charger
			api.ChargePhases
		}

		c = &PhasesDecorator{
			Charger:      c,
			ChargePhases: cp,
		}
	}

	return c, err
}

// NewConfigurable creates a new charger
func NewConfigurable(
	statusG func() (string, error),
	enabledG func() (bool, error),
	enableS func(bool) error,
	maxCurrentS func(int64) error,
) (api.Charger, error) {
	c := &Charger{
		statusG:     statusG,
		enabledG:    enabledG,
		enableS:     enableS,
		maxCurrentS: maxCurrentS,
	}

	return c, nil
}

// Status implements the Charger.Status interface
func (c *Charger) Status() (api.ChargeStatus, error) {
	s, err := c.statusG()
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(s), nil
}

// Enabled implements the Charger.Enabled interface
func (c *Charger) Enabled() (bool, error) {
	return c.enabledG()
}

// Enable implements the Charger.Enable interface
func (c *Charger) Enable(enable bool) error {
	return c.enableS(enable)
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (c *Charger) MaxCurrent(current int64) error {
	return c.maxCurrentS(current)
}

// ChargePhases decorates Charger with api.ChargePhases interface
type ChargePhases struct {
	phasesS func(int64) error
}

// NewChargePhases creates a new api.ChargePhases
func NewChargePhases(ccPhases provider.Config) (api.ChargePhases, error) {
	phasesS, err := provider.NewIntSetterFromConfig("phases", ccPhases)
	if err != nil {
		return nil, err
	}

	e := &ChargePhases{
		phasesS: phasesS,
	}

	return e, nil
}

// Phases1p3p implements api.ChargePhases interface
func (c *ChargePhases) Phases1p3p(phases int64) error {
	return c.phasesS(phases)
}
