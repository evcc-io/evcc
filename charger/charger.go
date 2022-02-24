package charger

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

// Charger is an api.Charger implementation with configurable getters and setters.
type Charger struct {
	statusG     func() (string, error)
	enabledG    func() (bool, error)
	enableS     func(bool) error
	maxCurrentS func(int64) error
}

func init() {
	registry.Add(api.Custom, NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates a new configurable charger
func NewConfigurableFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct{ Status, Enable, Enabled, MaxCurrent provider.Config }{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// TODO deprecate
	log := util.NewLogger("charger")
	cc.Status.Deprecate(log)
	cc.Enable.Deprecate(log)
	cc.Enabled.Deprecate(log)
	cc.MaxCurrent.Deprecate(log)

	status, err := provider.NewStringGetterFromConfig(cc.Status)
	if err != nil {
		return nil, fmt.Errorf("status: %w", err)
	}

	enabled, err := provider.NewBoolGetterFromConfig(cc.Enabled)
	if err != nil {
		return nil, fmt.Errorf("enabled: %w", err)
	}

	enable, err := provider.NewBoolSetterFromConfig("enable", cc.Enable)
	if err != nil {
		return nil, fmt.Errorf("enable: %w", err)
	}

	maxcurrent, err := provider.NewIntSetterFromConfig("maxcurrent", cc.MaxCurrent)
	if err != nil {
		return nil, fmt.Errorf("maxcurrent: %w", err)
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

// Status implements the api.Charger interface
func (m *Charger) Status() (api.ChargeStatus, error) {
	s, err := m.statusG()
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(s), nil
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
