package core

import (
	"time"

	"github.com/benbjohnson/clock"
	"github.com/jinzhu/now"
	"github.com/samber/lo"
)

type meterEnergy struct {
	clock     clock.Clock
	updated   time.Time
	startFunc func(time.Time) time.Time
	energy    *float64 // kWh
	acc       float64  // kWh
}

// reset previous period
func (m *meterEnergy) resetPeriod() {
	sod := m.startFunc(m.clock.Now())
	if m.startFunc != nil && m.updated.Before(sod) {
		m.updated = time.Time{}
		m.energy = nil
		m.acc = 0
	}
}

func (m *meterEnergy) AccumulatedEnergy() float64 {
	m.resetPeriod()
	return m.acc
}

func (m *meterEnergy) AddTotalEnergy(v float64) {
	m.resetPeriod()
	defer func() {
		m.updated = m.clock.Now()
		m.energy = lo.ToPtr(v)
	}()

	if m.energy == nil {
		return
	}

	m.acc += v - *m.energy
}

func (m *meterEnergy) AddPower(v float64) {
	m.resetPeriod()
	defer func() { m.updated = m.clock.Now() }()

	if m.updated.IsZero() {
		return
	}

	d := m.clock.Since(m.updated)
	m.acc += v * d.Hours() / 1e3
}

func beginningOfDay(t time.Time) time.Time {
	return now.With(t).BeginningOfDay()
}
