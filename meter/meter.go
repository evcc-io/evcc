package meter

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
)

// MeterEnergyDecorator decorates an api.Meter with api.MeterEnergy
type MeterEnergyDecorator struct {
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

	for k, v := range map[string]string{"power": cc.Power.Type} {
		if v == "" {
			log.FATAL.Fatalf("default meter config: %s required", k)
		}
	}

	m := NewConfigurable(provider.NewFloatGetterFromConfig(log, cc.Power))

	// decorate Meter with MeterEnergy
	if cc.Energy != nil {
		m = &MeterEnergyDecorator{
			Meter:       m,
			MeterEnergy: NewMeterEnergy(provider.NewFloatGetterFromConfig(log, *cc.Energy)),
		}
	}

	return m
}

// NewConfigurable creates a new charger
func NewConfigurable(currentPowerG func() (float64, error)) api.Meter {
	return &Meter{
		currentPowerG: currentPowerG,
	}
}

// Meter is an api.Meter implementation with configurable getters and setters.
type Meter struct {
	currentPowerG func() (float64, error)
}

// CurrentPower implements the Meter.CurrentPower interface
func (m *Meter) CurrentPower() (float64, error) {
	return m.currentPowerG()
}

// MeterEnergy is an api.MeterEnergy implementation with configurable getters and setters.
type MeterEnergy struct {
	totalEnergyG func() (float64, error)
}

// NewMeterEnergy creates a new charger
func NewMeterEnergy(totalEnergyG func() (float64, error)) api.MeterEnergy {
	return &MeterEnergy{
		totalEnergyG: totalEnergyG,
	}
}

// TotalEnergy implements the Meter.TotalEnergy interface
func (m *MeterEnergy) TotalEnergy() (float64, error) {
	return m.totalEnergyG()
}
