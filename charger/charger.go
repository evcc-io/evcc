package charger

import (
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
func NewConfigurableFromConfig(log *util.Logger, other map[string]interface{}) api.Charger {
	cc := struct{ Status, Enable, Enabled, MaxCurrent provider.Config }{}
	util.DecodeOther(log, other, &cc)

	for k, v := range map[string]string{
		"status":     cc.Status.Type,
		"enable":     cc.Enable.Type,
		"enabled":    cc.Enabled.Type,
		"maxcurrent": cc.MaxCurrent.Type,
	} {
		if v == "" {
			log.FATAL.Fatalf("default charger config: %s required", k)
		}
	}

	charger := NewConfigurable(
		provider.NewStringGetterFromConfig(log, cc.Status),
		provider.NewBoolGetterFromConfig(log, cc.Enabled),
		provider.NewBoolSetterFromConfig(log, "enable", cc.Enable),
		provider.NewIntSetterFromConfig(log, "maxcurrent", cc.MaxCurrent),
	)

	return charger
}

// NewConfigurable creates a new charger
func NewConfigurable(
	statusG func() (string, error),
	enabledG func() (bool, error),
	enableS func(bool) error,
	maxCurrentS func(int64) error,
) api.Charger {
	return &Charger{
		statusG:     statusG,
		enabledG:    enabledG,
		enableS:     enableS,
		maxCurrentS: maxCurrentS,
	}
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
