package meter

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
)

// CompositeMeter decorates a Meter with MeterEnergy.
type CompositeMeter struct {
	api.Meter
	api.MeterEnergy
}

// NewConfigurableFromConfig creates api.Meter from config
func NewConfigurableFromConfig(log *util.Logger, other map[string]interface{}) api.Meter {
	cc := struct {
		Power  provider.Config
		Energy *provider.Config // optional
	}{}
	util.DecodeOther(log, other, &cc)

	m := NewConfigurable(provider.NewFloatGetterFromConfig(log, cc.Power))

	// decorate Meter with MeterEnergy
	if cc.Energy != nil {
		m = &CompositeMeter{
			Meter:       m,
			MeterEnergy: NewMeterEnergy(provider.NewFloatGetterFromConfig(log, *cc.Energy)),
		}
	}

	return m
}

// NewConfigurable creates a new charger
func NewConfigurable(currentPowerG provider.FloatGetter) api.Meter {
	return &Meter{
		currentPowerG: currentPowerG,
	}
}

// Meter is an api.Meter implementation with configurable getters and setters.
type Meter struct {
	currentPowerG provider.FloatGetter
}

// CurrentPower implements the Meter.CurrentPower interface
func (m *Meter) CurrentPower() (float64, error) {
	return m.currentPowerG()
}

// MeterEnergy is an api.MeterEnergy implementation with configurable getters and setters.
type MeterEnergy struct {
	totalEnergyG provider.FloatGetter
}

// NewMeterEnergy creates a new charger
func NewMeterEnergy(totalEnergyG provider.FloatGetter) api.MeterEnergy {
	return &MeterEnergy{
		totalEnergyG: totalEnergyG,
	}
}

// TotalEnergy implements the Meter.TotalEnergy interface
func (m *MeterEnergy) TotalEnergy() (float64, error) {
	return m.totalEnergyG()
}
