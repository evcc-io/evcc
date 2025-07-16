package wrapper

import (
	"sync"
)

// ChargeMeter is a replacement for a physical charge meter.
// It uses the charger's actual or max current to calculate power consumption.
type ChargeMeter struct {
	mu    sync.Mutex
	power float64
}

// SetPower updates meter's current power
func (m *ChargeMeter) SetPower(power float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.power = power
}

// CurrentPower implements the api.Meter interface
func (m *ChargeMeter) CurrentPower() (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.power, nil
}
