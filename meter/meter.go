package meter

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add(api.Custom, NewConfigurableFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateMeter -b api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)" -t "api.MeterVoltage,Voltages,func() (float64, float64, float64, error)" -t "api.MeterPower,Powers,func() (float64, float64, float64, error)" -t "api.Battery,Soc,func() (float64, error)"

// NewConfigurableFromConfig creates api.Meter from config
func NewConfigurableFromConfig(other map[string]interface{}) (api.Meter, error) {
	var cc struct {
		Power    provider.Config
		Energy   *provider.Config  // optional
		Soc      *provider.Config  // optional
		Currents []provider.Config // optional
		Voltages []provider.Config // optional
		Powers   []provider.Config // optional
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	power, err := provider.NewFloatGetterFromConfig(cc.Power)
	if err != nil {
		return nil, fmt.Errorf("power: %w", err)
	}

	m, _ := NewConfigurable(power)

	// decorate Meter with MeterEnergy
	var totalEnergyG func() (float64, error)
	if cc.Energy != nil {
		totalEnergyG, err = provider.NewFloatGetterFromConfig(*cc.Energy)
		if err != nil {
			return nil, fmt.Errorf("energy: %w", err)
		}
	}

	// decorate Meter with MeterCurrent
	var currentsG func() (float64, float64, float64, error)
	if len(cc.Currents) > 0 {
		if len(cc.Currents) != 3 {
			return nil, errors.New("need 3 currents")
		}

		var curr []func() (float64, error)
		for idx, cc := range cc.Currents {
			c, err := provider.NewFloatGetterFromConfig(cc)
			if err != nil {
				return nil, fmt.Errorf("currents[%d]: %w", idx, err)
			}

			curr = append(curr, c)
		}

		currentsG = collectPhaseProviders(curr)
	}

	// decorate Meter with MeterVoltage
	var voltagesG func() (float64, float64, float64, error)
	if len(cc.Voltages) > 0 {
		if len(cc.Voltages) != 3 {
			return nil, errors.New("need 3 voltages")
		}

		var volt []func() (float64, error)
		for idx, cc := range cc.Voltages {
			c, err := provider.NewFloatGetterFromConfig(cc)
			if err != nil {
				return nil, fmt.Errorf("voltages[%d]: %w", idx, err)
			}

			volt = append(volt, c)
		}

		voltagesG = collectPhaseProviders(volt)
	}

	// decorate Meter with MeterPower
	var powersG func() (float64, float64, float64, error)
	if len(cc.Powers) > 0 {
		if len(cc.Powers) != 3 {
			return nil, errors.New("need 3 powers")
		}

		var pow []func() (float64, error)
		for idx, cc := range cc.Powers {
			c, err := provider.NewFloatGetterFromConfig(cc)
			if err != nil {
				return nil, fmt.Errorf("powers[%d]: %w", idx, err)
			}

			pow = append(pow, c)
		}

		powersG = collectPhaseProviders(pow)
	}

	// decorate Meter with BatterySoc
	var batterySocG func() (float64, error)
	if cc.Soc != nil {
		batterySocG, err = provider.NewFloatGetterFromConfig(*cc.Soc)
		if err != nil {
			return nil, fmt.Errorf("battery: %w", err)
		}
	}

	res := m.Decorate(totalEnergyG, currentsG, voltagesG, powersG, batterySocG)

	return res, nil
}

// collectPhaseProviders combines phase getters into currents api function
func collectPhaseProviders(g []func() (float64, error)) func() (float64, float64, float64, error) {
	return func() (float64, float64, float64, error) {
		var res []float64
		for _, currentG := range g {
			c, err := currentG()
			if err != nil {
				return 0, 0, 0, err
			}

			res = append(res, c)
		}

		return res[0], res[1], res[2], nil
	}
}

// NewConfigurable creates a new meter
func NewConfigurable(currentPowerG func() (float64, error)) (*Meter, error) {
	m := &Meter{
		currentPowerG: currentPowerG,
	}
	return m, nil
}

// Meter is an api.Meter implementation with configurable getters and setters.
type Meter struct {
	currentPowerG func() (float64, error)
}

// Decorate attaches additional capabilities to the base meter
func (m *Meter) Decorate(
	totalEnergy func() (float64, error),
	currents func() (float64, float64, float64, error),
	voltages func() (float64, float64, float64, error),
	powers func() (float64, float64, float64, error),
	batterySoc func() (float64, error),
) api.Meter {
	return decorateMeter(m, totalEnergy, currents, voltages, powers, batterySoc)
}

// CurrentPower implements the api.Meter interface
func (m *Meter) CurrentPower() (float64, error) {
	return m.currentPowerG()
}
