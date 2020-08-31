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
	phasesS     func(int64) error
}

func init() {
	registry.Add("default", NewConfigurableFromConfig)
}

// TODO generation of setters is not yet functional
// go:generate go run ../cmd/tools/decorate.go -p charger -f decorateCharger -b api.Charger -o charger_decorators -t "api.ChargePhases,Phases1p3p,func(int64) error"

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

	if err != nil {
		return nil, err
	}

	c, _ := NewConfigurable(status, enabled, enable, maxcurrent)

	var phases func(int64) error
	if cc.Phases != nil {
		if c.phasesS, err = provider.NewIntSetterFromConfig("phases", *cc.Phases); err != nil {
			return nil, err
		}
		phases = c.phases1p3p
	}

	return decorateCharger(c, phases), nil
}

// NewConfigurable creates a new charger
func NewConfigurable(
	statusG func() (string, error),
	enabledG func() (bool, error),
	enableS func(bool) error,
	maxCurrentS func(int64) error,
) (*Charger, error) {
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

// Phases1p3p implements the Charger.Phases1p3p interface
func (c *Charger) phases1p3p(phases int64) error {
	return c.phasesS(phases)
}
