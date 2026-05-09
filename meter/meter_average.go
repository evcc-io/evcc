package meter

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx("movingaverage", NewMovingAverageFromConfig)
}

// NewMovingAverageFromConfig creates api.Meter from config
func NewMovingAverageFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		Decay float64
		Meter struct {
			batteryCapacity `mapstructure:",squash"`
			Type            string
			Other           map[string]any `mapstructure:",remain"`
		}
	}{
		Decay: 0.1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	m, err := NewFromConfig(ctx, cc.Meter.Type, cc.Meter.Other)
	if err != nil {
		return nil, err
	}

	mav := &MovingAverage{
		decay:         cc.Decay,
		currentPowerG: m.CurrentPower,
	}

	meter, _ := NewConfigurable(mav.CurrentPower)

	// decorate import reading
	var importEnergy func() (float64, error)
	if m, ok := api.Cap[api.MeterImport](m); ok {
		importEnergy = m.ImportEnergy
	}

	// decorate export reading
	var exportEnergy func() (float64, error)
	if m, ok := api.Cap[api.MeterExport](m); ok {
		exportEnergy = m.ExportEnergy
	}

	// decorate battery reading
	var batterySoc func() (float64, error)
	if m, ok := api.Cap[api.Battery](m); ok {
		batterySoc = m.Soc
	}

	// decorate currents reading
	var currents func() (float64, float64, float64, error)
	if m, ok := api.Cap[api.PhaseCurrents](m); ok {
		currents = m.Currents
	}

	// decorate voltages reading
	var voltages func() (float64, float64, float64, error)
	if m, ok := api.Cap[api.PhaseVoltages](m); ok {
		voltages = m.Voltages
	}

	// decorate powers reading
	var powers func() (float64, float64, float64, error)
	if m, ok := api.Cap[api.PhasePowers](m); ok {
		powers = m.Powers
	}

	implement.May(meter, implement.MeterImport(importEnergy))
	implement.May(meter, implement.MeterExport(exportEnergy))

	if batterySoc != nil {
		implement.Has(meter, implement.Battery(batterySoc))
		implement.May(meter, implement.BatteryCapacity(cc.Meter.batteryCapacity.Decorator()))
		return meter, nil
	}

	implement.May(meter, implement.PhaseCurrents(currents))
	implement.May(meter, implement.PhaseVoltages(voltages))
	implement.May(meter, implement.PhasePowers(powers))

	return meter, nil
}

type MovingAverage struct {
	decay         float64
	value         *float64
	currentPowerG func() (float64, error)
}

func (m *MovingAverage) CurrentPower() (float64, error) {
	power, err := m.currentPowerG()
	if err != nil {
		return power, err
	}

	return m.add(power), nil
}

// modeled after https://github.com/VividCortex/ewma

// Add adds a value to the series and updates the moving average.
func (m *MovingAverage) add(value float64) float64 {
	if m.value == nil {
		m.value = &value
	} else {
		*m.value = (value * m.decay) + (*m.value * (1 - m.decay))
	}

	return *m.value
}
