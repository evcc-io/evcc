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

// NewConfigurableFromConfig creates a new configurable charger
func NewConfigurableFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct{ Status, Enable, Enabled, MaxCurrent provider.Config }{}
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

	return NewConfigurable(status, enabled, enable, maxcurrent)
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
func (m *Charger) Status() (api.ChargeStatus, error) {
	s, err := m.statusG()
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(s), nil
}

// Enabled implements the Charger.Enabled interface
func (m *Charger) Enabled() (bool, error) {
	return m.enabledG()
}

// Enable implements the Charger.Enable interface
func (m *Charger) Enable(enable bool) error {
	return m.enableS(enable)
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (m *Charger) MaxCurrent(current int64) error {
	return m.maxCurrentS(current)
}
