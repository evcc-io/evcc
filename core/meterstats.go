package core

import (
	"time"

	"github.com/jinzhu/now"
)

type meterStats struct {
	updated   time.Time
	startFunc func(time.Time) time.Time
	energy    float64 // kWh
}

// reset previous period
func (m *meterStats) resetPeriod(now time.Time) {
	if m.startFunc != nil && !m.updated.After(m.startFunc(now)) {
		m.energy = 0
	}
}

func (m *meterStats) Energy() float64 {
	m.resetPeriod(time.Now())
	return m.energy
}

func (m *meterStats) AddEnergy(v float64) {
	m.resetPeriod(time.Now())
	m.updated = time.Now()
	m.energy += v
}

func (m *meterStats) AddPower(v float64) {
	m.resetPeriod(time.Now())
	defer func() { m.updated = time.Now() }()

	if m.updated.IsZero() {
		return
	}

	m.energy += v * time.Since(m.updated).Hours() / 1e3
}

func beginningOfDay(t time.Time) time.Time {
	return now.With(t).BeginningOfDay()
}
