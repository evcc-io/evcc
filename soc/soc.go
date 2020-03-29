package soc

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

type embed struct {
	title    string
	capacity int64
}

// Title implements the SoC.Title interface
func (m *embed) Title() string {
	return m.title
}

// Capacity implements the SoC.Capacity interface
func (m *embed) Capacity() int64 {
	return m.capacity
}

// SoC is an api.SoC implementation with configurable getters and setters.
type SoC struct {
	*embed
	chargeG provider.FloatGetter
}

// NewConfigurableFromConfig creates a new SoC
func NewConfigurableFromConfig(log *api.Logger, other map[string]interface{}) api.SoC {
	cc := struct {
		Title    string
		Capacity int64
		Charge   *provider.Config
	}{}
	api.DecodeOther(log, other, &cc)

	return &SoC{
		embed:   &embed{cc.Title, cc.Capacity},
		chargeG: provider.NewFloatGetterFromConfig(cc.Charge),
	}
}

// ChargeState implements the SoC.ChargeState interface
func (m *SoC) ChargeState() (float64, error) {
	return m.chargeG()
}
