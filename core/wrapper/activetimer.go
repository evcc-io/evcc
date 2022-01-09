package wrapper

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
		clck:     clock.New(),
		duration: 0,
	}
}

// ActiveCharge signals active timer start
func (m *ActiveTimer) ActiveTimerHandler(lpEnabled bool, current float64) {
	if current > 0 && lpEnabled {
		m.StartActiveTimer()
	} else {
		m.StopActiveTimer()
	}
}

// ActiveCharge signals active timer start
func (m *ActiveTimer) StartActiveTimer() {
	m.Lock()
	defer m.Unlock()
	if !m.active {
		m.active = true
		m.start = m.clck.Now()
		m.duration = 0
	}
}

// StopCharge signals charge timer stop
func (m *ActiveTimer) StopActiveTimer() {
	m.Lock()
	defer m.Unlock()
	if m.active {
		m.active = false
		m.duration = m.clck.Since(m.start)
	}
}

// ActiveTime implements the api.ActiveTimer interface
func (m *ActiveTimer) ActiveTime() (time.Duration, error) {
	m.Lock()
	defer m.Unlock()

	if m.active {
		return m.clck.Since(m.start), nil
	}
	return m.duration, nil
}
