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

// go:generate go run ../cmd/tools/decorate.go -f decorateCustom -b *Charger -r api.Charger -t "api.Identifier,Identify,func() (string, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) (error)"

// NewConfigurableFromConfig creates a new configurable charger
func NewConfigurableFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		Status, Enable, Enabled, MaxCurrent provider.Config
		Identify, Phases1p3p                *provider.Config
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
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

	c, err := NewConfigurable(status, enabled, enable, maxcurrent)

	// decorator phases
	var phases1p3p func(int) error
	if err == nil && cc.Phases1p3p != nil {
		var phases1p3pi64 func(int64) error
		phases1p3pi64, err = provider.NewIntSetterFromConfig("phases", *cc.Phases1p3p)

		phases1p3p = func(phases int) error {
			return phases1p3pi64(int64(phases))
		}
	}

	// decorator identifier
	var identify func() (string, error)
	if err == nil && cc.Identify != nil {
		identify, err = provider.NewStringGetterFromConfig(*cc.Identify)
	}

	return decorateCustom(c, identify, phases1p3p), err
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
