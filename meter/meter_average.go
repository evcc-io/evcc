package meter

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("movingaverage", NewMovingAverageFromConfig)
}

// NewMovingAverageFromConfig creates api.Meter from config
func NewMovingAverageFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		Decay float64
		Meter struct {
			Type  string
			Other map[string]interface{} `mapstructure:",remain"`
		}
	}{
		Decay: 0.1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	m, err := NewFromConfig(cc.Meter.Type, cc.Meter.Other)
	if err != nil {
		return nil, err
	}

	mav := &MovingAverage{
		decay:         cc.Decay,
		currentPowerG: m.CurrentPower,
	}

	meter, _ := NewConfigurable(mav.CurrentPower)

	// decorate energy reading
	var totalEnergy func() (float64, error)
	if m, ok := m.(api.MeterEnergy); ok {
		totalEnergy = m.TotalEnergy
	}

	// decorate battery reading
	var batterySoc func() (float64, error)
	if m, ok := m.(api.Battery); ok {
		batterySoc = m.Soc
	}

	// decorate currents reading
	var currents func() (float64, float64, float64, error)
	if m, ok := m.(api.MeterCurrent); ok {
		currents = m.Currents
	}

	res := meter.Decorate(totalEnergy, currents, batterySoc)

	return res, nil
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
