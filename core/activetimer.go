package core

import (
	"sync"
	"time"

	"github.com/benbjohnson/clock"
)

// ActiveTimer measures active time between start and stop events
type ActiveTimer struct {
	sync.Mutex
	clck clock.Clock

	active   bool
	start    time.Time
	duration time.Duration
}

// NewActiveTimer creates ActiveTimer
// for tracking duration of an active Loadpoint
func NewActiveTimer() *ActiveTimer {
	return &ActiveTimer{
		clck: clock.New(),
	}
}

// start or stop the timer based on enabled and current
func (m *ActiveTimer) ActiveTimerHandler(lpEnabled bool, current float64) {
	if current > 0 && lpEnabled {
		m.StartActiveTimer()
	} else {
		m.StopActiveTimer()
	}
}

// start the active timer if not started already
func (m *ActiveTimer) StartActiveTimer() {
	m.Lock()
	defer m.Unlock()
	if !m.active {
		m.active = true
		m.start = m.clck.Now()
		m.duration = 0
	}
}

// stop the active timer and save the duration
func (m *ActiveTimer) StopActiveTimer() {
	m.Lock()
	defer m.Unlock()
	if m.active {
		m.active = false
		m.duration = m.clck.Since(m.start)
	}
}

// ActiveTime return the duration of the last started timer
func (m *ActiveTimer) ActiveTime() (time.Duration, error) {
	m.Lock()
	defer m.Unlock()

	if m.active {
		return m.clck.Since(m.start), nil
	}
	return m.duration, nil
}
