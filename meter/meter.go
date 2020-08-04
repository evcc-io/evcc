package meter

import (
	"errors"
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
)

// NewConfigurableFromConfig creates api.Meter from config
func NewConfigurableFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		Power    provider.Config
		Energy   *provider.Config  // optional
		Currents []provider.Config // optional
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
		energy, err := NewMeterEnergy(*cc.Energy)
		if err != nil {
			return nil, err
		}

		type EnergyDecorator struct {
			api.Meter
			api.MeterEnergy
		}

		m = &EnergyDecorator{
			Meter:       m,
			MeterEnergy: energy,
		}
	}

	// decorate Meter with MeterCurrent
	if len(cc.Currents) > 0 {
		currents, err := NewCurrents(cc.Currents)
		if err != nil {
			return nil, err
		}

		type PowerEnergy interface {
			api.Meter
			api.MeterEnergy
		}

		if pe, ok := m.(PowerEnergy); ok {
			type CurrentDecorator struct {
				PowerEnergy
				api.MeterCurrent
			}

			m = &CurrentDecorator{
				PowerEnergy:  pe,
				MeterCurrent: currents,
			}
		} else {
			type CurrentDecorator struct {
				api.Meter
				api.MeterCurrent
			}

			m = &CurrentDecorator{
				Meter:        m,
				MeterCurrent: currents,
			}
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

// NewMeterEnergy creates a new api.MeterEnergy
func NewMeterEnergy(ccEnergy provider.Config) (api.MeterEnergy, error) {
	totalEnergyG, err := provider.NewFloatGetterFromConfig(ccEnergy)
	if err != nil {
		return nil, err
	}

	e := &MeterEnergy{
		totalEnergyG: totalEnergyG,
	}

	return e, nil
}

// TotalEnergy implements the Meter.TotalEnergy interface
func (m *MeterEnergy) TotalEnergy() (float64, error) {
	return m.totalEnergyG()
}

// Currents is an api.MeterCurrent implementation
type Currents struct {
	currentsG []func() (float64, error)
}

// NewCurrents creates a new api.MeterCurrent
func NewCurrents(ccCurrents []provider.Config) (api.MeterCurrent, error) {
	if len(ccCurrents) != 3 {
		return nil, errors.New("need 3 currents")
	}

	var currentsG []func() (float64, error)
	for _, cc := range ccCurrents {
		c, err := provider.NewFloatGetterFromConfig(cc)
		if err != nil {
			return nil, err
		}

		currentsG = append(currentsG, c)
	}

	c := &Currents{
		currentsG: currentsG,
	}

	return c, nil
}

// Currents implements the api.Currents interface
func (c *Currents) Currents() (float64, float64, float64, error) {
	var currents []float64
	for _, currentG := range c.currentsG {
		c, err := currentG()
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, c)
	}

	return currents[0], currents[1], currents[2], nil
}
