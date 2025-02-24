package core

import (
	"time"

	"github.com/benbjohnson/clock"
	"github.com/jinzhu/now"
	"github.com/samber/lo"
)

type meterEnergy struct {
	clock       clock.Clock
	startFunc   func(time.Time) time.Time
	Updated     time.Time `json:"updated"`
	Meter       *float64  `json:"meter,omitempty"` // kWh
	Accumulated float64   `json:"accumulated"`     // kWh
}

// reset resets data
func (m *meterEnergy) reset() {
	m.Updated = time.Time{}
	m.Meter = nil
	m.Accumulated = 0
}

// resetPeriod resets data on period boundary
func (m *meterEnergy) resetPeriod() {
	sod := m.startFunc(m.clock.Now())
	if m.startFunc != nil && m.Updated.Before(sod) {
		m.reset()
	}
}

func (m *meterEnergy) AccumulatedEnergy() float64 {
	m.resetPeriod()
	return m.Accumulated
}

func (m *meterEnergy) AddMeterTotal(v float64) {
	m.resetPeriod()
	defer func() {
		m.Updated = m.clock.Now()
		m.Meter = lo.ToPtr(v)
	}()

	if m.Meter == nil {
		return
	}

	m.Accumulated += v - *m.Meter
}

func (m *meterEnergy) AddPower(v float64) {
	m.resetPeriod()
	defer func() { m.Updated = m.clock.Now() }()

	if m.Updated.IsZero() {
		return
	}

	d := m.clock.Since(m.Updated)
	m.Accumulated += v * d.Hours() / 1e3
}

func beginningOfDay(t time.Time) time.Time {
	return now.With(t).BeginningOfDay()
}
