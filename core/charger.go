package core

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// Charger is an api.Charger implementation with configurable getters and setters.
type Charger struct {
	statusG        provider.StringGetter
	actualCurrentG provider.IntGetter
	enabledG       provider.BoolGetter
	enableS        provider.BoolSetter
}

// NewCharger creates a new charger
func NewCharger(
	statusG provider.StringGetter,
	actualCurrentG provider.IntGetter,
	enabledG provider.BoolGetter,
	enableS provider.BoolSetter,
) api.Charger {
	return &Charger{
		statusG:        statusG,
		actualCurrentG: actualCurrentG,
		enabledG:       enabledG,
		enableS:        enableS,
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

// ActualCurrent implements the Charger.ActualCurrent interface
func (m *Charger) ActualCurrent() (int64, error) {
	return m.actualCurrentG()
}

// Enabled implements the Charger.Enabled interface
func (m *Charger) Enabled() (bool, error) {
	return m.enabledG()
}

// Enable implements the Charger.Enable interface
func (m *Charger) Enable(enable bool) error {
	return m.enableS(enable)
}
