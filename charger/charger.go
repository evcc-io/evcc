package charger

import (
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/server/config"
	"github.com/andig/evcc/util"
)

type genericConfig struct {
	Status     provider.Config `validate:"required" ui:"de=Ladestatus ('A'..'C')"`
	Enabled    provider.Config `validate:"required" ui:"de=Aktiviert (bool)"`
	Enable     provider.Config `validate:"required" ui:"de=Aktivieren (bool)"`
	MaxCurrent provider.Config `validate:"required" ui:"de=Ladestrom (A)"`
}

func init() {
	registry.Add("default", "Charger (frei konfigurierbar)", NewConfigurableFromConfig, genericConfig{}, config.LastRank)
}

// NewConfigurableFromConfig creates a new configurable charger
func NewConfigurableFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := genericConfig{}

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

// Charger is an api.Charger implementation with configurable getters and setters.
type Charger struct {
	statusG     func() (string, error)
	enabledG    func() (bool, error)
	enableS     func(bool) error
	maxCurrentS func(int64) error
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
