package wrapper

import (
	"sync"
)

// ChargeMeter is a replacement for a physical charge meter.
// It uses the charger's actual or max current to calculate power consumption.
type ChargeMeter struct {
	sync.Mutex
	Phases  int64
	Voltage float64
	power   float64
}

func currentToPower(current, voltage float64, phases int64) float64 {
	return float64(phases) * current * voltage
}

// SetChargeCurrent updates meter's current power based on actual current
func (m *ChargeMeter) SetChargeCurrent(current int64) {
	m.Lock()
	defer m.Unlock()
	m.power = currentToPower(float64(current), m.Voltage, m.Phases)
}

// CurrentPower implements the Meter.CurrentPower interface
func (m *ChargeMeter) CurrentPower() (float64, error) {
	m.Lock()
	defer m.Unlock()
	return m.power, nil
}
