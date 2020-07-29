package meter

import (
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
)

// EnergyDecorator decorates an api.Meter with api.MeterEnergy
type EnergyDecorator struct {
	api.Meter
	api.MeterEnergy
}

// NewConfigurableFromConfig creates api.Meter from config
func NewConfigurableFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		Power  provider.Config
		Energy *provider.Config // optional
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	for k, v := range map[string]string{"power": cc.Power.Type} {
		if v == "" {
			return nil, fmt.Errorf("default meter config: %s required", k)
		}
	}

	power, err := provider.NewFloatGetterFromConfig(cc.Power)
	if err != nil {
		return nil, err
	}

	m, _ := NewConfigurable(power)

	// decorate Meter with MeterEnergy
	if cc.Energy != nil {
		energy, err := provider.NewFloatGetterFromConfig(*cc.Energy)
		if err != nil {
			return nil, err
		}

		m = &EnergyDecorator{
			Meter:       m,
			MeterEnergy: NewMeterEnergy(energy),
		}
	}

	return m, nil
}

// NewConfigurable creates a new charger
func NewConfigurable(currentPowerG func() (float64, error)) (api.Meter, error) {
	m := &Meter{
		currentPowerG: currentPowerG,
	}
	return m, nil
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
