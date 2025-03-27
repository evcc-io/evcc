package core

import (
	"time"

	"github.com/benbjohnson/clock"
	"github.com/jinzhu/now"
	"github.com/samber/lo"
)

type meterEnergy struct {
	clock       clock.Clock
	updated     time.Time
	meter       *float64 // kWh
	Accumulated float64  `json:"accumulated"` // kWh
}

func (m *meterEnergy) AccumulatedEnergy() float64 {
	return m.Accumulated
}

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

func (m *meterEnergy) AddEnergy(v float64) {
	defer func() { m.updated = m.clock.Now() }()

	if m.updated.IsZero() {
		return
	}

	m.Accumulated += v
}

func (m *meterEnergy) AddPower(v float64) {
	m.AddEnergy(v * m.clock.Since(m.updated).Hours() / 1e3)
}

func beginningOfDay(t time.Time) time.Time {
	return now.With(t).BeginningOfDay()
}
