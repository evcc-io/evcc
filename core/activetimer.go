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

	started      time.Time
	lastduration time.Duration
}

// NewActiveTimer creates ActiveTimer
// for tracking duration of an active Loadpoint
func NewActiveTimer() *ActiveTimer {
	return &ActiveTimer{
		clck: clock.New(),
	}
}

// start the active timer if not started already
func (m *ActiveTimer) Start() {
	m.Lock()
	defer m.Unlock()
	if m.started.IsZero() {
		m.started = m.clck.Now()
	}
}

// stop and reset called
func (m *ActiveTimer) Reset() {
	m.Lock()
	defer m.Unlock()
	m.lastduration = 0
	m.started = time.Time{}
}

// stop the active timer and save the duration
func (m *ActiveTimer) Stop() {
	m.Lock()
	defer m.Unlock()
	if !m.started.IsZero() {
		m.lastduration = m.clck.Since(m.started)
		m.started = time.Time{}
	}
}

// returns the duration of the last started timer
func (m *ActiveTimer) duration() time.Duration {
	m.Lock()
	defer m.Unlock()

	if !m.started.IsZero() {
		return m.clck.Since(m.started)
	}
	return m.lastduration
}
