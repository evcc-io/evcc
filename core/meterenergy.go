package core

import (
	"time"

	"github.com/jinzhu/now"
	"github.com/samber/lo"
)

type meterEnergy struct {
	updated   time.Time
	startFunc func(time.Time) time.Time
	energy    *float64 // kWh
	acc       float64  // kWh
}

// reset previous period
func (m *meterEnergy) resetPeriod(now time.Time) {
	if m.startFunc != nil && !m.updated.After(m.startFunc(now)) {
		m.energy = nil
		m.acc = 0
	}
}

func (m *meterEnergy) AccumulatedEnergy() float64 {
	m.resetPeriod(time.Now())
	return m.acc
}

func (m *meterEnergy) AddTotalEnergy(v float64) {
	m.resetPeriod(time.Now())
	defer func() {
		m.updated = time.Now()
		m.energy = lo.ToPtr(v)
	}()

	if m.energy == nil {
		return
	}

	m.acc += v - *m.energy
}

func (m *meterEnergy) AddPower(v float64) {
	m.resetPeriod(time.Now())
	defer func() { m.updated = time.Now() }()

	if m.updated.IsZero() {
		return
	}

	m.acc += v * time.Since(m.updated).Hours() / 1e3
}

func beginningOfDay(t time.Time) time.Time {
	return now.With(t).BeginningOfDay()
}
