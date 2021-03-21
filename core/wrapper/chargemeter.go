package wrapper

import (
	"sync"
)

// ChargeMeter is a replacement for a physical charge meter.
// It uses the charger's actual or max current to calculate power consumption.
type ChargeMeter struct {
	sync.Mutex
	power float64
}

// SetPower updates meter's current power
func (m *ChargeMeter) SetPower(power float64) {
	m.Lock()
	defer m.Unlock()
	m.power = power
}

// CurrentPower implements the api.Meter interface
func (m *ChargeMeter) CurrentPower() (float64, error) {
	m.Lock()
	defer m.Unlock()
	return m.power, nil
}
