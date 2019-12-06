package core

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// SoC is an api.SoC implementation with configurable getters and setters.
type SoC struct {
	Capacity int64
	Title    string
	chargeG  provider.FloatGetter
}

// NewSoC creates a new SoC
func NewSoC(capacity int64, title string, chargeG provider.FloatGetter) api.SoC {
	return &SoC{
		Capacity: capacity,
		Title:    title,
		chargeG:  chargeG,
	}
}

// ChargeState implements the SoC.ChargeState interface
func (m *SoC) ChargeState() (float64, error) {
	return m.chargeG()
}
