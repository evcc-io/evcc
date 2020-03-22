package charger

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// Charger is an api.Charger implementation with configurable getters and setters.
type Charger struct {
	statusG     provider.StringGetter
	enabledG    provider.BoolGetter
	enableS     provider.BoolSetter
	maxCurrentS provider.IntSetter
}

type config struct {
	Status, Enable, Enabled, MaxCurrent *provider.Config
}

// NewChargerFromConfig creates a new configurable charger
func NewChargerFromConfig(log *api.Logger, other map[string]interface{}) api.Charger {
	var cc config
	decodeOther(log, other, &cc)

	charger := NewCharger(
		provider.NewStringGetterFromConfig(cc.Status),
		provider.NewBoolGetterFromConfig(cc.Enabled),
		provider.NewBoolSetterFromConfig("enable", cc.Enable),
		provider.NewIntSetterFromConfig("current", cc.MaxCurrent),
	)

	return charger
}

// NewCharger creates a new charger
func NewCharger(
	statusG provider.StringGetter,
	enabledG provider.BoolGetter,
	enableS provider.BoolSetter,
	maxCurrentS provider.IntSetter,
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

// MaxCurrent implements the Charger.MaxCurrent API
func (m *Charger) MaxCurrent(current int64) error {
	return m.maxCurrentS(current)
}
