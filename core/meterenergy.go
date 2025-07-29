package core

import (
	"time"

	"github.com/benbjohnson/clock"
	"github.com/samber/lo"
)

type meterEnergy struct {
	clock       clock.Clock
	updated     time.Time
	meter       *float64 // kWh
	Accumulated float64  `json:"accumulated"` // kWh
}

// AccumulatedEnergy returns the accumulated energy in kWh
func (m *meterEnergy) AccumulatedEnergy() float64 {
	return m.Accumulated
}

// AddMeterTotal adds the difference to the last total meter value in kWh
func (m *meterEnergy) AddMeterTotal(v float64) {
	defer func() {
		m.updated = m.clock.Now()
		m.meter = lo.ToPtr(v)
	}()

	if m.meter == nil {
		return
	}

	m.Accumulated += v - *m.meter
}

// AddEnergy adds the given energy in kWh
func (m *meterEnergy) AddEnergy(v float64) {
	defer func() { m.updated = m.clock.Now() }()

	if m.updated.IsZero() {
		return
	}

	m.Accumulated += v
}

// AddPower adds the given power in W, calculating the energy based on the time since the last update
func (m *meterEnergy) AddPower(v float64) {
	m.AddEnergy(v * m.clock.Since(m.updated).Hours() / 1e3)
}
