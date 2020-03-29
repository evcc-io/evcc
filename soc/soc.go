package soc

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// SoC is an api.SoC implementation with configurable getters and setters.
type SoC struct {
	capacity int64
	title    string
	chargeG  provider.FloatGetter
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
		title:    cc.Title,
		capacity: cc.Capacity,
		chargeG:  provider.NewFloatGetterFromConfig(cc.Charge),
	}
}

// Title implements the SoC.Title interface
func (m *SoC) Title() string {
	return m.title
}

// Capacity implements the SoC.Capacity interface
func (m *SoC) Capacity() int64 {
	return m.capacity
}

// ChargeState implements the SoC.ChargeState interface
func (m *SoC) ChargeState() (float64, error) {
	return m.chargeG()
}
